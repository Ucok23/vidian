package git

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RepoProfile is a deterministic, no-AI snapshot of a repo's shape: what stack
// it's built with, where to start reading, and what's most active. It is the
// factual input an AI narrator can later summarize — the AI never invents any
// of this, it only explains it.
type RepoProfile struct {
	Stack       []StackEntry `json:"stack"`
	EntryPoints []EntryPoint `json:"entryPoints"`
	KeyFiles    []KeyFile    `json:"keyFiles"`
	Stats       RepoStats    `json:"stats"`
}

type StackEntry struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

type EntryPoint struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

type KeyFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

var mainGoPattern = regexp.MustCompile(`(^|/)main\.go$`)

// GetRepoProfile analyzes dir's tracked files and commit history to build a
// deterministic RepoProfile. No file contents are read except small manifest
// files needed to detect frameworks (e.g. package.json dependencies).
func GetRepoProfile(dir string) (RepoProfile, error) {
	profile := RepoProfile{
		Stack:       []StackEntry{},
		EntryPoints: []EntryPoint{},
		KeyFiles:    []KeyFile{},
	}

	out, err := RunGitCommand(dir, "ls-files")
	if err != nil {
		return profile, err
	}
	files := []string{}
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}

	profile.Stack = detectStack(dir, files)
	profile.EntryPoints = detectEntryPoints(files)

	hotFiles, err := GetHotFiles(dir, 5)
	if err != nil {
		hotFiles = []HotFile{}
	}
	profile.KeyFiles = buildKeyFiles(files, hotFiles)

	stats, err := GetRepoStats(dir)
	if err != nil {
		stats = RepoStats{}
	}
	profile.Stats = stats

	return profile, nil
}

// findManifests returns every tracked file whose base name matches name,
// shallowest path first — manifests live at any depth in monorepos (e.g. a
// frontend/package.json alongside a root go.mod).
func findManifests(files []string, name string) []string {
	matches := []string{}
	for _, f := range files {
		if filepath.Base(f) == name {
			matches = append(matches, f)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return strings.Count(matches[i], "/") < strings.Count(matches[j], "/")
	})
	return matches
}

// detectStack inspects tracked files for language/framework manifests at any
// depth, since monorepos commonly nest a frontend/ package.json alongside a
// root go.mod.
func detectStack(dir string, files []string) []StackEntry {
	stack := []StackEntry{}

	for _, m := range findManifests(files, "go.mod") {
		detail := m
		if mod := readModuleName(filepath.Join(dir, m)); mod != "" {
			detail = mod
		}
		stack = append(stack, StackEntry{Name: "Go", Detail: detail})
	}

	for _, m := range findManifests(files, "package.json") {
		name, frameworks := readPackageJSON(filepath.Join(dir, m))
		detail := m
		if name != "" {
			detail = name
		}
		stack = append(stack, StackEntry{Name: "Node/JavaScript", Detail: detail})
		for _, fw := range frameworks {
			stack = append(stack, StackEntry{Name: fw, Detail: m})
		}
	}

	for _, m := range findManifests(files, "Cargo.toml") {
		stack = append(stack, StackEntry{Name: "Rust", Detail: m})
	}
	for _, m := range findManifests(files, "requirements.txt") {
		stack = append(stack, StackEntry{Name: "Python", Detail: m})
	}
	for _, m := range findManifests(files, "pyproject.toml") {
		stack = append(stack, StackEntry{Name: "Python", Detail: m})
	}
	for _, m := range findManifests(files, "Gemfile") {
		stack = append(stack, StackEntry{Name: "Ruby", Detail: m})
	}
	for _, m := range findManifests(files, "pom.xml") {
		stack = append(stack, StackEntry{Name: "Java", Detail: m})
	}

	return stack
}

// readModuleName extracts the module path from the first line of a go.mod file.
func readModuleName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

var knownFrameworks = map[string]string{
	"svelte": "Svelte", "react": "React", "vue": "Vue",
	"@angular/core": "Angular", "next": "Next.js", "nuxt": "Nuxt",
}

// readPackageJSON returns the package name and any recognized frontend
// frameworks found in dependencies/devDependencies.
func readPackageJSON(path string) (name string, frameworks []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	var pkg struct {
		Name            string            `json:"name"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", nil
	}
	seen := make(map[string]bool)
	for dep := range pkg.Dependencies {
		if fw, ok := knownFrameworks[dep]; ok && !seen[fw] {
			frameworks = append(frameworks, fw)
			seen[fw] = true
		}
	}
	for dep := range pkg.DevDependencies {
		if fw, ok := knownFrameworks[dep]; ok && !seen[fw] {
			frameworks = append(frameworks, fw)
			seen[fw] = true
		}
	}
	return pkg.Name, frameworks
}

// detectEntryPoints finds conventional entry-point files by path pattern.
func detectEntryPoints(files []string) []EntryPoint {
	entries := []EntryPoint{}
	present := make(map[string]bool, len(files))
	for _, f := range files {
		present[f] = true
	}

	for _, f := range files {
		if mainGoPattern.MatchString(f) {
			entries = append(entries, EntryPoint{Path: f, Label: "Go entry point"})
		}
	}

	for _, candidate := range []string{"index.html", "src/main.js", "src/main.ts", "src/App.svelte"} {
		if present[candidate] {
			entries = append(entries, EntryPoint{Path: candidate, Label: "Frontend entry point"})
		}
	}
	for _, f := range files {
		if strings.HasSuffix(f, "/index.html") && !strings.Contains(f, "node_modules") {
			entries = append(entries, EntryPoint{Path: f, Label: "Frontend entry point"})
		}
	}

	if present["Dockerfile"] {
		entries = append(entries, EntryPoint{Path: "Dockerfile", Label: "Container build"})
	}
	if present["Makefile"] {
		entries = append(entries, EntryPoint{Path: "Makefile", Label: "Build entry point"})
	}

	return entries
}

var readmePattern = regexp.MustCompile(`(?i)^readme(\.md|\.txt)?$`)

// buildKeyFiles ranks README, manifests, and the most frequently changed
// files (hot files) as the best starting points for reading the codebase.
func buildKeyFiles(files []string, hotFiles []HotFile) []KeyFile {
	keyFiles := []KeyFile{}
	seen := make(map[string]bool)

	add := func(path, reason string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		keyFiles = append(keyFiles, KeyFile{Path: path, Reason: reason})
	}

	for _, f := range files {
		if readmePattern.MatchString(filepath.Base(f)) {
			add(f, "README")
		}
	}

	manifests := map[string]bool{
		"go.mod": true, "package.json": true, "Cargo.toml": true,
		"requirements.txt": true, "pyproject.toml": true, "Gemfile": true, "pom.xml": true,
	}
	for _, f := range files {
		if manifests[filepath.Base(f)] {
			add(f, "manifest")
		}
	}

	for _, hf := range hotFiles {
		add(hf.Path, "frequently changed")
	}

	return keyFiles
}
