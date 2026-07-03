// Package ai narrates a deterministic git.RepoProfile into prose. It is
// strictly a phrasing layer: the model is given only the facts already
// computed by internal/git and instructed not to add any it wasn't given.
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

// apiURL is a var (not const) so tests can point it at an httptest.Server.
var apiURL = "https://api.anthropic.com/v1/messages"

const model = "claude-sonnet-4-5"

var ErrNoAPIKey = errors.New("no Anthropic API key configured")

const systemPrompt = `You are writing a short onboarding tour for a codebase, for a developer who has never seen it before.

You will be given a JSON object with facts already extracted from the repository: its detected stack, entry points, key files, and activity stats. Use ONLY the facts given to you. Do not invent file names, libraries, or behavior that isn't in the JSON. Do not speculate about what the code does beyond what the facts imply.

Write 3-4 short paragraphs in plain prose (no headers, no bullet lists):
1. What this project is and what stack it's built with.
2. How it's laid out and where to start reading (reference the entry points and key files by their exact paths).
3. A closing note on activity/maturity if the stats are informative (commit count, contributors, streak).

Keep it concise and factual. If a category is empty, don't mention it.`

// Narrate sends a RepoProfile to Claude and returns a prose tour grounded
// strictly in that profile. apiKey must be non-empty.
func Narrate(profile git.RepoProfile, apiKey string) (string, error) {
	if apiKey == "" {
		return "", ErrNoAPIKey
	}

	factsJSON, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("marshal profile: %w", err)
	}

	reqBody := map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": string(factsJSON)},
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
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to Anthropic API failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API returned %d: %s", resp.StatusCode, string(respBytes))
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
