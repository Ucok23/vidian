package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestHandleGitNarrativeMissingWorkspace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/git/narrative", nil)
	w := httptest.NewRecorder()
	handleGitNarrative(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing workspace param, got %d", w.Code)
	}
}
