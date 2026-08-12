// Episode 12 — build, release, reproducibility. Runs the actual clean-tree
// checks (go build, go vet, a CGO-free build) into a scratch directory so
// nothing is left behind in the workspace, plus text-based checks over
// go.mod / Makefile / CI config for -trimpath, toolchain pinning, and how
// version info is surfaced.
package goreview

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// CommandResult is the outcome of one real command run against the module.
type CommandResult struct {
	Ran     bool   `json:"ran"`
	OK      bool   `json:"ok"`
	Output  string `json:"output,omitempty"`
	Command string `json:"command"`
}

// BuildReport is the full episode-12 artifact.
type BuildReport struct {
	Build              CommandResult `json:"build"`
	Vet                CommandResult `json:"vet"`
	CGOFreeBuild       CommandResult `json:"cgoFreeBuild"`
	TrimpathConfigured bool          `json:"trimpathConfigured"`
	TrimpathFoundIn    []string      `json:"trimpathFoundIn"`
	ToolchainLine      string        `json:"toolchainLine,omitempty"`
	UsesReadBuildInfo  bool          `json:"usesReadBuildInfo"`
	UsesLdflagsVersion bool          `json:"usesLdflagsVersion"`
	HasDockerfile      bool          `json:"hasDockerfile"`
	DockerFindings     []string      `json:"dockerFindings"`
}

func runCmd(dir, name string, env []string, args ...string) CommandResult {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	return CommandResult{
		Ran:     true,
		OK:      err == nil,
		Output:  string(out),
		Command: name + " " + strings.Join(args, " "),
	}
}

// runBuildReview implements episode 12. All real builds are redirected into
// a temp directory so no binaries land in the workspace tree.
func runBuildReview(dir string, _ ModuleInfo) BuildReport {
	var report BuildReport

	// Three independent `go` invocations over the whole module — each one
	// can take a while on a real codebase, so they run concurrently instead
	// of back to back.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		tmp, err := os.MkdirTemp("", "vidian-goreview-build-*")
		if err == nil {
			defer os.RemoveAll(tmp)
			report.Build = runCmd(dir, "go", nil, "build", "-o", tmp+string(os.PathSeparator), "./...")
		}
	}()
	go func() {
		defer wg.Done()
		report.Vet = runCmd(dir, "go", nil, "vet", "./...")
	}()
	go func() {
		defer wg.Done()
		tmp2, err := os.MkdirTemp("", "vidian-goreview-cgofree-*")
		if err == nil {
			defer os.RemoveAll(tmp2)
			report.CGOFreeBuild = runCmd(dir, "go", []string{"CGO_ENABLED=0"}, "build", "-o", tmp2+string(os.PathSeparator), "./...")
		}
	}()
	wg.Wait()

	// -trimpath: a convention check across the files that actually invoke
	// the build, not the source tree itself.
	buildFiles := []string{"Makefile", "makefile", "Dockerfile", ".goreleaser.yml", ".goreleaser.yaml"}
	filepath.Walk(filepath.Join(dir, ".github", "workflows"), func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			buildFiles = append(buildFiles, rel)
		}
		return nil
	})
	for _, f := range buildFiles {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		text := string(data)
		if strings.Contains(text, "-trimpath") {
			report.TrimpathConfigured = true
			report.TrimpathFoundIn = append(report.TrimpathFoundIn, f)
		}
		if strings.Contains(text, "-ldflags") && (strings.Contains(text, "version") || strings.Contains(text, "Version")) {
			report.UsesLdflagsVersion = true
		}
	}

	// toolchain pinning, from go.mod directly.
	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "toolchain ") {
				report.ToolchainLine = strings.TrimPrefix(line, "toolchain ")
			}
		}
	}

	// runtime/debug.ReadBuildInfo usage anywhere in internal source.
	readBuildInfoRe := regexp.MustCompile(`debug\.ReadBuildInfo`)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil && readBuildInfoRe.Match(data) {
			report.UsesReadBuildInfo = true
		}
		return nil
	})

	// Dockerfile checks.
	if data, err := os.ReadFile(filepath.Join(dir, "Dockerfile")); err == nil {
		report.HasDockerfile = true
		text := string(data)
		lastFrom := ""
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
				lastFrom = trimmed
			}
		}
		if lastFrom != "" {
			lower := strings.ToLower(lastFrom)
			if strings.Contains(lower, "scratch") || strings.Contains(lower, "distroless") {
				report.DockerFindings = append(report.DockerFindings, "Final stage: "+lastFrom+" (minimal base — good)")
			} else {
				report.DockerFindings = append(report.DockerFindings, "Final stage: "+lastFrom+" — not scratch/distroless, check whether that's intentional")
			}
		}
		if !strings.Contains(text, "USER ") {
			report.DockerFindings = append(report.DockerFindings, "No USER directive found — image likely runs as root")
		}
		if dockerignore, err := os.ReadFile(filepath.Join(dir, ".dockerignore")); err != nil || !strings.Contains(string(dockerignore), ".git") {
			report.DockerFindings = append(report.DockerFindings, ".dockerignore missing or doesn't exclude .git")
		}
	}

	return report
}
