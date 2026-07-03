package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ucok23/vidian/internal/git"
)

func withMockAPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	prev := apiURL
	apiURL = server.URL
	t.Cleanup(func() {
		server.Close()
		apiURL = prev
	})
}

func TestNarrateNoAPIKey(t *testing.T) {
	_, err := Narrate(git.RepoProfile{}, "")
	if err != ErrNoAPIKey {
		t.Fatalf("expected ErrNoAPIKey, got %v", err)
	}
}

func TestNarrateSendsProfileAndAuthHeaders(t *testing.T) {
	profile := git.RepoProfile{
		Stack: []git.StackEntry{{Name: "Go", Detail: "github.com/example/thing"}},
	}

	var gotBody map[string]any
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("expected x-api-key header 'test-key', got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Error("expected anthropic-version header to be set")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "This is a Go project."}},
		})
	})

	narrative, err := Narrate(profile, "test-key")
	if err != nil {
		t.Fatalf("Narrate returned error: %v", err)
	}
	if narrative != "This is a Go project." {
		t.Errorf("expected narrative text from mock response, got %q", narrative)
	}

	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) == 0 {
		t.Fatalf("expected non-empty messages in request body, got %+v", gotBody)
	}
	firstMsg, _ := messages[0].(map[string]any)
	content, _ := firstMsg["content"].(string)
	if !strings.Contains(content, "github.com/example/thing") {
		t.Errorf("expected request content to include the repo profile facts, got %q", content)
	}

	system, _ := gotBody["system"].(string)
	if !strings.Contains(system, "Use ONLY the facts given") {
		t.Errorf("expected system prompt to instruct grounding in given facts, got %q", system)
	}
}

func TestNarrateAPIError(t *testing.T) {
	withMockAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	})

	_, err := Narrate(git.RepoProfile{}, "bad-key")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention status code, got %v", err)
	}
}
