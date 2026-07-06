package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteOpenAIHitsChatCompletions(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "hello from openai"}},
			},
		})
	}))
	defer server.Close()

	p := Provider{Kind: "openai", BaseURL: server.URL + "/v1/", Model: "test-model", APIKey: "sk-test"}
	out, err := p.Complete("sys", "user", 128)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if out != "hello from openai" {
		t.Errorf("unexpected content: %q", out)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("expected /v1/chat/completions (trailing slash trimmed), got %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("expected bearer auth header, got %q", gotAuth)
	}
	if gotBody["model"] != "test-model" {
		t.Errorf("expected model passed through, got %v", gotBody["model"])
	}
	msgs, _ := gotBody["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected system+user messages, got %+v", gotBody["messages"])
	}
}

func TestCompleteOpenAIRequiresBaseAndModel(t *testing.T) {
	if _, err := (Provider{Kind: "openai", Model: "m"}).Complete("s", "u", 10); err == nil {
		t.Error("expected error when base URL is missing")
	}
	if _, err := (Provider{Kind: "openai", BaseURL: "http://x"}).Complete("s", "u", 10); err == nil {
		t.Error("expected error when model is missing")
	}
}

func TestCompleteAnthropicNoKey(t *testing.T) {
	if _, err := (Provider{Kind: "anthropic"}).Complete("s", "u", 10); err != ErrNoAPIKey {
		t.Errorf("expected ErrNoAPIKey, got %v", err)
	}
}

func TestExplainSendsCodeAndFilename(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "explained"}}},
		})
	}))
	defer server.Close()

	p := Provider{Kind: "openai", BaseURL: server.URL, Model: "m"}
	out, err := Explain("func F() {}", "main.go", p)
	if err != nil {
		t.Fatalf("Explain error: %v", err)
	}
	if out != "explained" {
		t.Errorf("unexpected output %q", out)
	}
	msgs, _ := gotBody["messages"].([]any)
	user, _ := msgs[len(msgs)-1].(map[string]any)
	content, _ := user["content"].(string)
	if !strings.Contains(content, "main.go") || !strings.Contains(content, "func F()") {
		t.Errorf("expected filename and code in user message, got %q", content)
	}
}
