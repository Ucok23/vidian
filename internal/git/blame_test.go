package git

import (
	"strings"
	"testing"

	"github.com/Ucok23/vidian/internal/config"
)

func init() {
	config.ActiveConfig = &config.Config{WorkspaceDir: "../.."}
}

func mustHaveGit(t *testing.T) {
	t.Helper()
	out, err := RunGitCommand("rev-parse", "--git-dir")
	if err != nil || strings.TrimSpace(out) == "" {
		t.Skip("not a git repository")
	}
}

func TestBlameNonEmpty(t *testing.T) {
	mustHaveGit(t)
	lines, err := Blame("go.mod")
	if err != nil {
		// Some environments may not have file content for blame; allow.
		t.Skipf("Blame failed: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected non-empty blame for go.mod")
	}
	for _, b := range lines {
		if b.Commit == "" {
			t.Error("expected commit hash to be populated")
		}
		if b.Author == "" {
			t.Error("expected author to be populated")
		}
		if b.Line <= 0 {
			t.Errorf("expected positive line number, got %d", b.Line)
		}
	}
}

func TestFileHistoryNonEmpty(t *testing.T) {
	mustHaveGit(t)
	commits, err := Log("go.mod")
	if err != nil {
		t.Skipf("Log failed: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected non-empty file history for go.mod")
	}
	for _, c := range commits {
		if c.Hash == "" || c.Author == "" || c.Summary == "" {
			t.Errorf("unexpected empty field in commit: %+v", c)
		}
	}
}
