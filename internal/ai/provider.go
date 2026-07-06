package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Ucok23/vidian/internal/git"
)

// Provider describes which chat backend to call. Kind "anthropic" uses the
// Anthropic Messages API; Kind "openai" uses any OpenAI-compatible
// /chat/completions endpoint (OpenAI itself, Ollama, LM Studio, vLLM, …),
// which is what lets users point Vidian at a local or self-hosted model
// instead of a cloud key; Kind "gemini" uses Google's native
// generateContent API.
type Provider struct {
	Kind    string // "anthropic" | "openai" | "gemini"
	BaseURL string // openai: API root (…/v1); gemini: API root (…/v1beta)
	APIKey  string
	Model   string
}

// Default models / endpoints used when a provider doesn't specify one.
const (
	DefaultAnthropicModel = "claude-sonnet-5"
	DefaultGeminiModel    = "gemini-2.0-flash"
	DefaultGeminiBaseURL  = "https://generativelanguage.googleapis.com/v1beta"
)

// Complete sends a single system+user turn and returns the assistant's text.
// It dispatches on p.Kind so callers (Narrate, Explain) don't care which
// backend is configured.
func (p Provider) Complete(systemPrompt, userContent string, maxTokens int) (string, error) {
	switch p.Kind {
	case "openai":
		return p.completeOpenAI(systemPrompt, userContent, maxTokens)
	case "gemini":
		return p.completeGemini(systemPrompt, userContent, maxTokens)
	case "anthropic", "":
		return p.completeAnthropic(systemPrompt, userContent, maxTokens)
	default:
		return "", fmt.Errorf("unknown AI provider %q", p.Kind)
	}
}

func (p Provider) completeAnthropic(systemPrompt, userContent string, maxTokens int) (string, error) {
	if p.APIKey == "" {
		return "", ErrNoAPIKey
	}
	model := p.Model
	if model == "" {
		model = DefaultAnthropicModel
	}
	reqBody := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userContent},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	respBytes, err := doRequest(req)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	for _, block := range parsed.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", errors.New("no text content in Anthropic response")
}

func (p Provider) completeOpenAI(systemPrompt, userContent string, maxTokens int) (string, error) {
	if p.BaseURL == "" {
		return "", errors.New("no base URL configured for OpenAI-compatible provider")
	}
	if p.Model == "" {
		return "", errors.New("no model configured for OpenAI-compatible provider")
	}
	reqBody := map[string]any{
		"model":      p.Model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := p.BaseURL
	for len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	url += "/chat/completions"

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	respBytes, err := doRequest(req)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("no choices in OpenAI-compatible response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func (p Provider) completeGemini(systemPrompt, userContent string, maxTokens int) (string, error) {
	if p.APIKey == "" {
		return "", ErrNoAPIKey
	}
	model := p.Model
	if model == "" {
		model = DefaultGeminiModel
	}
	base := p.BaseURL
	if base == "" {
		base = DefaultGeminiBaseURL
	}
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}

	reqBody := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": userContent}}},
		},
		"generationConfig": map[string]any{"maxOutputTokens": maxTokens},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", base, model)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// Header auth keeps the key out of the URL (and out of any request logs).
	req.Header.Set("x-goog-api-key", p.APIKey)

	respBytes, err := doRequest(req)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	for _, c := range parsed.Candidates {
		for _, part := range c.Content.Parts {
			if part.Text != "" {
				return part.Text, nil
			}
		}
	}
	return "", errors.New("no text content in Gemini response")
}

// doRequest executes an already-built request and returns the body, mapping
// non-2xx responses to an error that names the status.
func doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI API returned %d: %s", resp.StatusCode, string(respBytes))
	}
	return respBytes, nil
}

// NarrateWith produces an onboarding tour like Narrate, but through an
// arbitrary Provider (Anthropic or OpenAI-compatible).
func NarrateWith(profile git.RepoProfile, p Provider) (string, error) {
	factsJSON, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("marshal profile: %w", err)
	}
	return p.Complete(systemPrompt, string(factsJSON), 1024)
}

const explainSystemPrompt = `You are helping a developer understand a single source file (or a selected snippet) they are reading in a code viewer. You will be given the file path and its contents.

Explain, grounded strictly in what you are shown:
1. What this code is and the role it plays.
2. The key functions/types/exports and how they fit together — reference them by name.
3. Anything worth flagging: side effects, external calls, notable control flow, or gotchas.

Keep it concise and concrete. Use short paragraphs; a short bullet list is fine for enumerating symbols. Do not invent behavior, files, or libraries that aren't present in what you were given.`

// Explain returns a prose explanation of the given code. filename is used only
// to give the model context about what it's looking at.
func Explain(code, filename string, p Provider) (string, error) {
	const maxChars = 16000
	if len(code) > maxChars {
		code = code[:maxChars] + "\n… (truncated)"
	}
	user := fmt.Sprintf("File: %s\n\n```\n%s\n```", filename, code)
	return p.Complete(explainSystemPrompt, user, 1024)
}
