// Episode 14 — automate everything mechanical. Vidian is a read-only
// viewer, so this never writes a config to disk: if the repo already has a
// golangci-lint config it's shown as-is (the policy document the course
// describes); if not, a justified baseline is suggested as text to copy in,
// not silently written. golangci-lint itself only runs if it's already
// installed — same graceful-skip convention as govulncheck in episode 08.
package goreview

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LintReport is the full episode-14 artifact.
type LintReport struct {
	ConfigExists          bool     `json:"configExists"`
	ConfigPath            string   `json:"configPath,omitempty"`
	ConfigContent         string   `json:"configContent,omitempty"`
	SuggestedConfig       string   `json:"suggestedConfig,omitempty"`
	GolangciLintAvailable bool     `json:"golangciLintAvailable"`
	LintOutput            string   `json:"lintOutput,omitempty"`
	LintError             string   `json:"lintError,omitempty"`
	LintTimedOut          bool     `json:"lintTimedOut"`
	IssueCount            int      `json:"issueCount"`
	GofmtFiles            []string `json:"gofmtFiles"`
	CIRunsLint            bool     `json:"ciRunsLint"`
	CIUsesNewFromRev      bool     `json:"ciUsesNewFromRev"`
}

var configNames = []string{".golangci.yml", ".golangci.yaml", ".golangci.toml", ".golangci.json"}

// suggestedGolangciConfig mirrors the baseline the course lays out in
// episode 14, each linter with a one-line justification as a comment —
// "the config is reviewable as policy" only if the reasons are in it.
const suggestedGolangciConfig = `linters:
  enable:
    - errcheck      # unchecked error returns — the highest-value linter in Go
    - govet         # stdlib correctness checks
    - staticcheck   # the largest and best analyzer set
    - ineffassign   # assignments never read
    - unused        # dead code
    - bodyclose     # unclosed HTTP response bodies
    - rowserrcheck  # unchecked sql.Rows.Err
    - sqlclosecheck # unclosed sql.Rows / sql.Stmt
    - noctx         # HTTP requests without context
    - contextcheck  # functions that should take a context
    - errorlint     # %v where %w was meant, err == sentinel instead of errors.Is
    - exhaustive    # non-exhaustive switch on enum types

# gocyclo/funlen/lll are deliberately not enabled here — the course's advice
# is to set their thresholds at the 90th percentile of *this* codebase
# rather than a book number, which this deterministic tool can't decide for
# you. Measure first, then add them with a repo-specific threshold.
`

func runLintReview(dir string, _ ModuleInfo) LintReport {
	var report LintReport

	for _, name := range configNames {
		path := filepath.Join(dir, name)
		if data, err := os.ReadFile(path); err == nil {
			report.ConfigExists = true
			report.ConfigPath = name
			report.ConfigContent = string(data)
			break
		}
	}
	if !report.ConfigExists {
		report.SuggestedConfig = suggestedGolangciConfig
	}

	report.GofmtFiles = findGofmtFiles(dir)

	if path, err := exec.LookPath("golangci-lint"); err == nil {
		report.GolangciLintAvailable = true
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		args := []string{"run", "./..."}
		var cleanupCfg string
		if !report.ConfigExists {
			// Run against the suggested baseline without ever writing into
			// the user's repo — a temp file, outside the tree, removed after.
			tmp, err := os.CreateTemp("", "vidian-goreview-golangci-*.yml")
			if err == nil {
				tmp.WriteString(suggestedGolangciConfig)
				tmp.Close()
				cleanupCfg = tmp.Name()
				args = append(args, "--config", tmp.Name())
			}
		}

		cmd := exec.CommandContext(ctx, path, args...)
		cmd.Dir = dir
		out, runErr := cmd.CombinedOutput()
		if cleanupCfg != "" {
			os.Remove(cleanupCfg)
		}
		report.LintOutput = string(out)
		if ctx.Err() == context.DeadlineExceeded {
			report.LintTimedOut = true
		} else if runErr != nil {
			if _, ok := runErr.(*exec.ExitError); !ok {
				report.LintError = runErr.Error()
			}
		}
		report.IssueCount = countLintIssues(report.LintOutput)
	}

	ciDir := filepath.Join(dir, ".github", "workflows")
	filepath.Walk(ciDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(data)
		if strings.Contains(text, "golangci-lint") {
			report.CIRunsLint = true
			if strings.Contains(text, "new-from-rev") {
				report.CIUsesNewFromRev = true
			}
		}
		return nil
	})

	return report
}

// findGofmtFiles runs `gofmt -l` — the files that would change under
// gofmt, i.e. not currently formatted. Read-only: -l lists, never -w writes.
func findGofmtFiles(dir string) []string {
	cmd := exec.Command("gofmt", "-l", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return []string{}
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "vendor/") || strings.Contains(line, "/vendor/") {
			continue
		}
		files = append(files, line)
	}
	return files
}

var lintIssueLineRe = regexp.MustCompile(`(?m)^\S+\.go:\d+:\d+:`)

func countLintIssues(output string) int {
	return len(lintIssueLineRe.FindAllString(output, -1))
}
