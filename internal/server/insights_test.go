package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Ucok23/vidian/internal/config"
	"github.com/Ucok23/vidian/internal/git"
)

var testWS *config.Workspace

func init() {
	config.ActiveConfig = config.New(false, 0)
	abs, _ := filepath.Abs("../..")
	testWS = config.ActiveConfig.Add(abs)
}

func TestHandleGitActivity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/git/activity?ws="+testWS.ID, nil)
	w := httptest.NewRecorder()

	handleGitActivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var counts map[string]int
	if err := json.NewDecoder(w.Body).Decode(&counts); err != nil {
		t.Fatalf("response is not valid JSON map[string]int: %v", err)
	}

	// Every key must be a YYYY-MM-DD string; every value must be positive.
	for date, n := range counts {
		if len(date) != 10 || date[4] != '-' || date[7] != '-' {
			t.Errorf("key %q does not look like YYYY-MM-DD", date)
		}
		if n <= 0 {
			t.Errorf("date %q has non-positive count %d", date, n)
		}
	}
}

func TestHandleGitHotFiles(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/git/hot-files?ws="+testWS.ID, nil)
	w := httptest.NewRecorder()

	handleGitHotFiles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var files []git.HotFile
	if err := json.NewDecoder(w.Body).Decode(&files); err != nil {
		t.Fatalf("response is not valid JSON []git.HotFile: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one hot file in response")
	}

	// Sorted descending.
	for i := 1; i < len(files); i++ {
		if files[i].Commits > files[i-1].Commits {
			t.Errorf("files not sorted: index %d (%d commits) > index %d (%d commits)",
				i, files[i].Commits, i-1, files[i-1].Commits)
		}
	}
}

func TestHandleGitActivityCORSOptions(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/git/activity", nil)
	w := httptest.NewRecorder()

	handleGitActivity(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("OPTIONS preflight: expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected CORS header '*', got %q", got)
	}
}
