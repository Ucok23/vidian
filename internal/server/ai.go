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

// defaultOpenAIBaseURL is used for the hosted "openai" provider when the user
// doesn't override it (the local "openai-compatible" provider always requires
// an explicit base URL).
const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// providerFromSettings resolves the active AI backend from the per-provider
// config map. It returns ok=false when the active provider isn't set up with
// the fields it needs.
func providerFromSettings(s *config.Settings) (ai.Provider, bool) {
	if s == nil {
		return ai.Provider{}, false
	}
	cfg := s.Providers[s.ActiveProvider]
	switch s.ActiveProvider {
	case "anthropic":
		if cfg.APIKey == "" {
			return ai.Provider{}, false
		}
		return ai.Provider{Kind: "anthropic", APIKey: cfg.APIKey, Model: cfg.Model}, true
	case "openai":
		if cfg.APIKey == "" || cfg.Model == "" {
			return ai.Provider{}, false
		}
		base := cfg.BaseURL
		if base == "" {
			base = defaultOpenAIBaseURL
		}
		return ai.Provider{Kind: "openai", BaseURL: base, Model: cfg.Model, APIKey: cfg.APIKey}, true
	case "openai-compatible":
		if cfg.BaseURL == "" || cfg.Model == "" {
			return ai.Provider{}, false
		}
		return ai.Provider{Kind: "openai", BaseURL: cfg.BaseURL, Model: cfg.Model, APIKey: cfg.APIKey}, true
	case "gemini":
		if cfg.APIKey == "" {
			return ai.Provider{}, false
		}
		return ai.Provider{Kind: "gemini", BaseURL: cfg.BaseURL, Model: cfg.Model, APIKey: cfg.APIKey}, true
	}
	return ai.Provider{}, false
}

// settingsStatus is the non-secret view of settings returned to the client:
// the active provider, whether it's usable, and per-provider non-secret config
// plus a hasKey flag so the UI can prefill fields and show which are set up.
func settingsStatus(s *config.Settings) map[string]any {
	_, ok := providerFromSettings(s)
	provs := map[string]any{}
	for _, id := range config.KnownProviders {
		cfg := s.Providers[id]
		provs[id] = map[string]any{
			"baseUrl": cfg.BaseURL,
			"model":   cfg.Model,
			"hasKey":  cfg.APIKey != "",
		}
	}
	return map[string]any{
		"activeProvider": s.ActiveProvider,
		"configured":     ok,
		"providers":      provs,
		"hasKey":         s.Providers["anthropic"].APIKey != "", // legacy field
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
	// "set to this value, possibly clearing it" (non-nil). The client sends
	// { activeProvider, provider: {...} } to configure one provider and make
	// it active; anthropicApiKey is a legacy shortcut kept for compatibility.
	var body struct {
		ActiveProvider *string `json:"activeProvider"`
		Provider       *struct {
			BaseURL *string `json:"baseUrl"`
			Model   *string `json:"model"`
			APIKey  *string `json:"apiKey"`
		} `json:"provider"`
		AnthropicAPIKey *string `json:"anthropicApiKey"` // legacy
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	settings, err := config.LoadSettings()
	if err != nil {
		settings = &config.Settings{}
	}
	if settings.Providers == nil {
		settings.Providers = map[string]config.ProviderConfig{}
	}

	// Legacy shortcut: a bare anthropicApiKey sets and activates Anthropic.
	if body.AnthropicAPIKey != nil {
		cfg := settings.Providers["anthropic"]
		cfg.APIKey = *body.AnthropicAPIKey
		settings.Providers["anthropic"] = cfg
		settings.ActiveProvider = "anthropic"
	}

	if body.ActiveProvider != nil {
		id := *body.ActiveProvider
		if !config.IsKnownProvider(id) {
			http.Error(w, "unknown provider", http.StatusBadRequest)
			return
		}
		settings.ActiveProvider = id
		if body.Provider != nil {
			cfg := settings.Providers[id]
			if body.Provider.BaseURL != nil {
				cfg.BaseURL = *body.Provider.BaseURL
			}
			if body.Provider.Model != nil {
				cfg.Model = *body.Provider.Model
			}
			if body.Provider.APIKey != nil {
				cfg.APIKey = *body.Provider.APIKey
			}
			settings.Providers[id] = cfg
		}
	}

	// Blank the legacy scalar fields now that config lives in the map, so they
	// don't shadow it on the next load/migrate.
	settings.AnthropicAPIKey = ""
	settings.AIProvider = ""
	settings.AIBaseURL = ""
	settings.AIModel = ""
	settings.AIAPIKey = ""

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
