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
	json.NewEncoder(w).Encode(map[string]bool{"hasKey": settings.AnthropicAPIKey != ""})
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
	var body struct {
		AnthropicAPIKey string `json:"anthropicApiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := config.SaveSettings(&config.Settings{AnthropicAPIKey: body.AnthropicAPIKey}); err != nil {
		http.Error(w, "failed to save settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"hasKey": body.AnthropicAPIKey != ""})
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
	if err != nil || settings.AnthropicAPIKey == "" {
		http.Error(w, "no Anthropic API key configured", http.StatusBadRequest)
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

	narrative, err := ai.Narrate(profile, settings.AnthropicAPIKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	narrativeCacheMu.Lock()
	narrativeCache[cacheKey] = narrative
	narrativeCacheMu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"narrative": narrative})
}
