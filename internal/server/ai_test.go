package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ucok23/vidian/internal/config"
)

// withTempSettings redirects config.LoadSettings/SaveSettings (which use
// os.UserConfigDir) to a throwaway directory for the duration of the test.
func withTempSettings(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

func TestHandleSettingsRoundTrip(t *testing.T) {
	withTempSettings(t)

	// No key saved yet.
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handleGetSettings(w, req)
	var got map[string]bool
	json.NewDecoder(w.Body).Decode(&got)
	if got["hasKey"] {
		t.Fatal("expected hasKey=false before any key is saved")
	}

	// Save a key.
	req = httptest.NewRequest(http.MethodPost, "/api/settings/save", strings.NewReader(`{"anthropicApiKey":"sk-ant-test"}`))
	w = httptest.NewRecorder()
	handleSaveSettings(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// hasKey should now be true.
	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w = httptest.NewRecorder()
	handleGetSettings(w, req)
	json.NewDecoder(w.Body).Decode(&got)
	if !got["hasKey"] {
		t.Fatal("expected hasKey=true after saving a key")
	}
}

func TestHandleGitNarrativeNoKey(t *testing.T) {
	withTempSettings(t)

	req := httptest.NewRequest(http.MethodGet, "/api/git/narrative?ws="+testWS.ID, nil)
	w := httptest.NewRecorder()
	handleGitNarrative(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 with no API key configured, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProviderFromSettingsPerProvider(t *testing.T) {
	cases := []struct {
		name     string
		s        config.Settings
		wantOK   bool
		wantKind string
	}{
		{"anthropic ok", config.Settings{ActiveProvider: "anthropic", Providers: map[string]config.ProviderConfig{"anthropic": {APIKey: "k"}}}, true, "anthropic"},
		{"anthropic no key", config.Settings{ActiveProvider: "anthropic", Providers: map[string]config.ProviderConfig{"anthropic": {}}}, false, ""},
		{"openai needs model", config.Settings{ActiveProvider: "openai", Providers: map[string]config.ProviderConfig{"openai": {APIKey: "k"}}}, false, ""},
		{"openai ok", config.Settings{ActiveProvider: "openai", Providers: map[string]config.ProviderConfig{"openai": {APIKey: "k", Model: "gpt"}}}, true, "openai"},
		{"compat needs base", config.Settings{ActiveProvider: "openai-compatible", Providers: map[string]config.ProviderConfig{"openai-compatible": {Model: "m"}}}, false, ""},
		{"compat ok", config.Settings{ActiveProvider: "openai-compatible", Providers: map[string]config.ProviderConfig{"openai-compatible": {BaseURL: "http://x/v1", Model: "m"}}}, true, "openai"},
		{"gemini ok", config.Settings{ActiveProvider: "gemini", Providers: map[string]config.ProviderConfig{"gemini": {APIKey: "AIza"}}}, true, "gemini"},
		{"none active", config.Settings{}, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, ok := providerFromSettings(&c.s)
			if ok != c.wantOK {
				t.Fatalf("ok=%v, want %v", ok, c.wantOK)
			}
			if ok && p.Kind != c.wantKind {
				t.Errorf("kind=%q, want %q", p.Kind, c.wantKind)
			}
		})
	}
	// The hosted OpenAI provider fills in a default base URL.
	p, _ := providerFromSettings(&config.Settings{ActiveProvider: "openai", Providers: map[string]config.ProviderConfig{"openai": {APIKey: "k", Model: "gpt"}}})
	if p.BaseURL != defaultOpenAIBaseURL {
		t.Errorf("expected default openai base url, got %q", p.BaseURL)
	}
}

func TestHandleSaveSettingsMultiProvider(t *testing.T) {
	withTempSettings(t)

	// Configure Gemini and make it active.
	req := httptest.NewRequest(http.MethodPost, "/api/settings/save",
		strings.NewReader(`{"activeProvider":"gemini","provider":{"model":"gemini-2.0-flash","apiKey":"AIza-x"}}`))
	w := httptest.NewRecorder()
	handleSaveSettings(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("save failed: %d %s", w.Code, w.Body.String())
	}
	var got map[string]any
	json.NewDecoder(w.Body).Decode(&got)
	if got["activeProvider"] != "gemini" {
		t.Errorf("expected active gemini, got %v", got["activeProvider"])
	}
	if got["configured"] != true {
		t.Errorf("expected configured=true, got %v", got["configured"])
	}

	// Re-reading should report the gemini key present and it still active.
	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w = httptest.NewRecorder()
	handleGetSettings(w, req)
	json.NewDecoder(w.Body).Decode(&got)
	provs, _ := got["providers"].(map[string]any)
	gem, _ := provs["gemini"].(map[string]any)
	if gem["hasKey"] != true {
		t.Errorf("expected gemini hasKey=true, got %+v", gem)
	}
	if gem["model"] != "gemini-2.0-flash" {
		t.Errorf("expected model persisted, got %+v", gem)
	}
}

func TestHandleGitNarrativeMissingWorkspace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/git/narrative", nil)
	w := httptest.NewRecorder()
	handleGitNarrative(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing workspace param, got %d", w.Code)
	}
}
