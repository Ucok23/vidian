package git

import (
	"strings"
	"testing"
)

func TestGetRepoProfile(t *testing.T) {
	mustHaveGit(t)
	profile, err := GetRepoProfile(testDir)
	if err != nil {
		t.Fatalf("GetRepoProfile returned error: %v", err)
	}

	// Vidian's own repo has a go.mod at the root, so Go must be detected.
	foundGo := false
	for _, s := range profile.Stack {
		if s.Name == "Go" {
			foundGo = true
			if !strings.HasPrefix(s.Detail, "github.com/") && s.Detail != "go.mod" {
				t.Errorf("expected Go module detail to look like a module path, got %q", s.Detail)
			}
		}
	}
	if !foundGo {
		t.Error("expected Go to be detected in stack")
	}

	// cmd/vidian/main.go should be picked up as a Go entry point.
	foundEntry := false
	for _, e := range profile.EntryPoints {
		if e.Path == "cmd/vidian/main.go" {
			foundEntry = true
		}
	}
	if !foundEntry {
		t.Errorf("expected cmd/vidian/main.go to be detected as an entry point, got %+v", profile.EntryPoints)
	}

	// README.md should be ranked as a key file.
	foundReadme := false
	for _, k := range profile.KeyFiles {
		if k.Path == "README.md" && k.Reason == "README" {
			foundReadme = true
		}
	}
	if !foundReadme {
		t.Errorf("expected README.md to be detected as a key file, got %+v", profile.KeyFiles)
	}

	if profile.Stats.TotalCommits == 0 {
		t.Error("expected non-zero total commits in profile stats")
	}

	// Vidian is a monorepo: frontend/package.json is nested, not at the root.
	// Node and Svelte must still be detected there (name comes from the
	// package.json "name" field when present, path otherwise).
	foundNode, foundSvelte := false, false
	for _, s := range profile.Stack {
		if s.Name == "Node/JavaScript" && s.Detail == "frontend" {
			foundNode = true
		}
		if s.Name == "Svelte" {
			foundSvelte = true
		}
	}
	if !foundNode {
		t.Errorf("expected Node/JavaScript to be detected from nested frontend/package.json, got %+v", profile.Stack)
	}
	if !foundSvelte {
		t.Errorf("expected Svelte to be detected as a framework, got %+v", profile.Stack)
	}
}

func TestReadPackageJSONFrameworks(t *testing.T) {
	name, frameworks := readPackageJSON("../../frontend/package.json")
	if name == "" {
		t.Fatal("expected package name to be read from frontend/package.json")
	}
	found := false
	for _, fw := range frameworks {
		if fw == "Svelte" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Svelte to be detected as a dependency, got %+v", frameworks)
	}
}
