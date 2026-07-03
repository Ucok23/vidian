package config

import (
	"path/filepath"
	"testing"
)

func withTempSettingsPath(t *testing.T) {
	t.Helper()
	prev := settingsPathOverride
	settingsPathOverride = filepath.Join(t.TempDir(), "settings.json")
	t.Cleanup(func() { settingsPathOverride = prev })
}

func TestLoadSettingsMissingFile(t *testing.T) {
	withTempSettingsPath(t)

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings returned error for missing file: %v", err)
	}
	if s.AnthropicAPIKey != "" {
		t.Errorf("expected empty key for missing settings file, got %q", s.AnthropicAPIKey)
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	withTempSettingsPath(t)

	want := &Settings{AnthropicAPIKey: "sk-ant-test-123"}
	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	got, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}
	if got.AnthropicAPIKey != want.AnthropicAPIKey {
		t.Errorf("expected key %q, got %q", want.AnthropicAPIKey, got.AnthropicAPIKey)
	}
}

func TestSaveSettingsOverwrites(t *testing.T) {
	withTempSettingsPath(t)

	if err := SaveSettings(&Settings{AnthropicAPIKey: "first"}); err != nil {
		t.Fatalf("first SaveSettings failed: %v", err)
	}
	if err := SaveSettings(&Settings{AnthropicAPIKey: "second"}); err != nil {
		t.Fatalf("second SaveSettings failed: %v", err)
	}

	got, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}
	if got.AnthropicAPIKey != "second" {
		t.Errorf("expected overwritten key %q, got %q", "second", got.AnthropicAPIKey)
	}
}
