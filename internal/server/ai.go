package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Ucok23/vidian/internal/ai"
	"github.com/Ucok23/vidian/internal/config"
	"github.com/Ucok23/vidian/internal/git"
)

// narrativeCache holds generated onboarding narratives keyed by
// "<workspace-id>@<head-commit>" so reopening a workspace at the same commit
// doesn't re-spend API tokens. It resets on server restart; that's an
// accepted v1 tradeoff over persisting it to disk.
var (
	narrativeCacheMu sync.Mutex
	narrativeCache   = map[string]string{}
)

// providerFromSettings resolves the effective AI backend. It returns ok=false
// when nothing usable is configured. An explicit "openai" provider needs a
// base URL and model; otherwise an Anthropic key is used if present.
func providerFromSettings(s *config.Settings) (ai.Provider, bool) {
	if s == nil {
		return ai.Provider{}, false
	}
	if s.AIProvider == "openai" {
		if s.AIBaseURL == "" || s.AIModel == "" {
			return ai.Provider{}, false
		}
		return ai.Provider{Kind: "openai", BaseURL: s.AIBaseURL, Model: s.AIModel, APIKey: s.AIAPIKey}, true
	}
	if s.AnthropicAPIKey != "" {
		return ai.Provider{Kind: "anthropic", APIKey: s.AnthropicAPIKey, Model: s.AIModel}, true
	}
	return ai.Provider{}, false
}

// settingsStatus is the non-secret view of settings returned to the client.
func settingsStatus(s *config.Settings) map[string]any {
	_, ok := providerFromSettings(s)
	return map[string]any{
		"hasKey":     s.AnthropicAPIKey != "", // legacy field: Anthropic key present
		"aiProvider": s.AIProvider,
		"aiBaseUrl":  s.AIBaseURL,
		"aiModel":    s.AIModel,
		"hasAiKey":   s.AIAPIKey != "",
		"configured": ok,
	}
}

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	settings, err := config.LoadSettings()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]bool{"hasKey": false})
		return
	}
	json.NewEncoder(w).Encode(settingsStatus(settings))
}

func handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Pointer fields distinguish "omitted, keep existing" (nil) from
	// "set to this value, possibly clearing it" (non-nil).
	var body struct {
		AnthropicAPIKey *string `json:"anthropicApiKey"`
		AIProvider      *string `json:"aiProvider"`
		AIBaseURL       *string `json:"aiBaseUrl"`
		AIModel         *string `json:"aiModel"`
		AIAPIKey        *string `json:"aiApiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	settings, err := config.LoadSettings()
	if err != nil {
		settings = &config.Settings{}
	}
	if body.AnthropicAPIKey != nil {
		settings.AnthropicAPIKey = *body.AnthropicAPIKey
	}
	if body.AIProvider != nil {
		settings.AIProvider = *body.AIProvider
	}
	if body.AIBaseURL != nil {
		settings.AIBaseURL = *body.AIBaseURL
	}
	if body.AIModel != nil {
		settings.AIModel = *body.AIModel
	}
	if body.AIAPIKey != nil {
		settings.AIAPIKey = *body.AIAPIKey
	}
	if err := config.SaveSettings(settings); err != nil {
		http.Error(w, "failed to save settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settingsStatus(settings))
}

// handleAIExplain returns a prose explanation of a file (or a selected snippet).
// The client posts { path, code } — code is the current buffer/selection so we
// explain exactly what the user sees, without re-reading from disk.
func handleAIExplain(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ws := resolveWorkspace(w, r)
	if ws == nil {
		return
	}
	var body struct {
		Path string `json:"path"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Code == "" {
		http.Error(w, "no code provided", http.StatusBadRequest)
		return
	}

	settings, _ := config.LoadSettings()
	provider, ok := providerFromSettings(settings)
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		http.Error(w, "no AI provider configured", http.StatusBadRequest)
		return
	}

	label := body.Path
	if label == "" {
		label = "(snippet)"
	}
	explanation, err := ai.Explain(body.Code, label, provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"explanation": explanation})
}

func handleGitNarrative(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	ws := resolveWorkspace(w, r)
	if ws == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")

	settings, err := config.LoadSettings()
	if err != nil {
		settings = &config.Settings{}
	}
	provider, ok := providerFromSettings(settings)
	if !ok {
		http.Error(w, "no AI provider configured", http.StatusBadRequest)
		return
	}

	head, err := git.RunGitCommand(ws.Path, "rev-parse", "HEAD")
	if err != nil {
		http.Error(w, "failed to resolve current commit", http.StatusInternalServerError)
		return
	}
	cacheKey := ws.ID + "@" + head
	regenerate := r.URL.Query().Get("regenerate") == "1"

	if !regenerate {
		narrativeCacheMu.Lock()
		cached, ok := narrativeCache[cacheKey]
		narrativeCacheMu.Unlock()
		if ok {
			json.NewEncoder(w).Encode(map[string]string{"narrative": cached})
			return
		}
	}

	profile, err := git.GetRepoProfile(ws.Path)
	if err != nil {
		http.Error(w, "failed to build repo profile", http.StatusInternalServerError)
		return
	}

	narrative, err := ai.NarrateWith(profile, provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	narrativeCacheMu.Lock()
	narrativeCache[cacheKey] = narrative
	narrativeCacheMu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"narrative": narrative})
}
