package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSettingsFile points settings I/O at a temp file containing raw JSON.
func writeSettingsFile(t *testing.T, raw string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	prev := settingsPathOverride
	settingsPathOverride = path
	t.Cleanup(func() { settingsPathOverride = prev })
}

func TestMigrateLegacyAnthropicKey(t *testing.T) {
	writeSettingsFile(t, `{"anthropicApiKey":"sk-ant-legacy","aiModel":"claude-x"}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.ActiveProvider != "anthropic" {
		t.Errorf("expected active provider anthropic, got %q", s.ActiveProvider)
	}
	got := s.Providers["anthropic"]
	if got.APIKey != "sk-ant-legacy" {
		t.Errorf("legacy key not migrated, got %q", got.APIKey)
	}
	if got.Model != "claude-x" {
		t.Errorf("legacy model not migrated, got %q", got.Model)
	}
}

func TestMigrateLegacyOpenAICompatible(t *testing.T) {
	writeSettingsFile(t, `{"aiProvider":"openai","aiBaseUrl":"http://localhost:11434/v1","aiModel":"llama3","aiApiKey":"k"}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.ActiveProvider != "openai-compatible" {
		t.Errorf("expected active provider openai-compatible, got %q", s.ActiveProvider)
	}
	got := s.Providers["openai-compatible"]
	if got.BaseURL != "http://localhost:11434/v1" || got.Model != "llama3" || got.APIKey != "k" {
		t.Errorf("legacy openai config not migrated: %+v", got)
	}
}

func TestNewFormatLoadsUnchanged(t *testing.T) {
	writeSettingsFile(t, `{"activeProvider":"gemini","providers":{"gemini":{"model":"gemini-2.0-flash","apiKey":"AIza"}}}`)
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.ActiveProvider != "gemini" {
		t.Errorf("expected gemini, got %q", s.ActiveProvider)
	}
	if s.Providers["gemini"].APIKey != "AIza" {
		t.Errorf("gemini key not loaded: %+v", s.Providers["gemini"])
	}
}
