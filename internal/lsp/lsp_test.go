package lsp

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsPathLike(t *testing.T) {
	// Candidates are authored with "/", but Windows workspace roots arrive with
	// "\"; both must count as explicit paths (not bare $PATH names).
	pathLike := []string{
		"{ws}/node_modules/.bin/x",
		`C:\Users\u\proj\x`,
		"~/.cargo/bin/rust-analyzer",
		"/usr/bin/gopls",
	}
	for _, c := range pathLike {
		if !isPathLike(c) {
			t.Errorf("isPathLike(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"gopls", "clangd", "typescript-language-server"} {
		if isPathLike(c) {
			t.Errorf("isPathLike(%q) = true, want false", c)
		}
	}
}

func TestExpandHomeTilde(t *testing.T) {
	got := expandHome("~/go/bin/gopls")
	want := filepath.Join(homeDir(), "go", "bin", "gopls")
	if got != want {
		t.Errorf("expandHome = %q, want %q", got, want)
	}
	if expandHome("gopls") != "gopls" {
		t.Errorf("bare name should be unchanged")
	}
}

func TestExpandPlaceholdersUsesHostSeparators(t *testing.T) {
	got := expandPlaceholders("{ws}/node_modules/.bin/x", filepath.Join("home", "u", "proj"))
	want := filepath.Join("home", "u", "proj", "node_modules", ".bin", "x")
	if got != want {
		t.Errorf("expandPlaceholders = %q, want %q", got, want)
	}
}

func TestExecutableCandidates(t *testing.T) {
	got := executableCandidates(filepath.Join("dir", "gopls"))
	if got[0] != filepath.Join("dir", "gopls") {
		t.Errorf("first candidate should be the path as-is, got %q", got[0])
	}
	if runtime.GOOS == "windows" {
		if len(got) < 2 {
			t.Errorf("Windows should append PATHEXT variants, got %v", got)
		}
	} else if len(got) != 1 {
		t.Errorf("non-Windows should return only the path, got %v", got)
	}
}
