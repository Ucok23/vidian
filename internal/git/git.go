package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Ucok23/vidian/internal/config"
)

type GitInfo struct {
	IsGit         bool     `json:"isGit"`
	CurrentBranch string   `json:"currentBranch"`
	Branches      []string `json:"branches"`
}

type GitChange struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type BlameLine struct {
	Line    int    `json:"line"`
	Commit  string `json:"commit"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Summary string `json:"summary"`
}

type CommitInfo struct {
	Hash     string `json:"hash"`
	Author   string `json:"author"`
	Email    string `json:"email"`
	Date     string `json:"date"`
	Relative string `json:"relative"`
	Summary  string `json:"summary"`
}

func RunGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = config.ActiveConfig.WorkspaceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func CheckIsGit() bool {
	_, err := exec.LookPath("git")
	if err != nil {
		return false
	}
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = config.ActiveConfig.WorkspaceDir
	return cmd.Run() == nil
}

func GetBranches() (GitInfo, error) {
	if !CheckIsGit() {
		return GitInfo{IsGit: false}, nil
	}

	currentBranch, err := RunGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		currentBranch = "HEAD"
	}

	branchesOut, err := RunGitCommand("branch", "-a", "--format=%(refname:short)")
	branches := []string{}
	if err == nil {
		lines := strings.Split(branchesOut, "\n")
		seen := make(map[string]bool)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "->") || strings.Contains(line, "HEAD") {
				continue
			}
			if strings.HasPrefix(line, "remotes/") {
				line = strings.TrimPrefix(line, "remotes/")
			}
			if !seen[line] {
				seen[line] = true
				branches = append(branches, line)
			}
		}
	}

	return GitInfo{
		IsGit:         true,
		CurrentBranch: currentBranch,
		Branches:      branches,
	}, nil
}

func Checkout(branch string) error {
	// Validate branch name
	branchesOut, err := RunGitCommand("branch", "-a", "--format=%(refname:short)")
	if err != nil {
		return fmt.Errorf("failed to fetch branches: %w", err)
	}

	isValid := false
	lines := strings.Split(branchesOut, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "remotes/") {
			line = strings.TrimPrefix(line, "remotes/")
		}
		if line == branch {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid branch name: %s", branch)
	}

	_, err = RunGitCommand("checkout", branch)
	return err
}

func GetChanges() ([]GitChange, error) {
	statusOut, err := RunGitCommand("status", "--porcelain")
	if err != nil {
		return []GitChange{}, err
	}

	changes := []GitChange{}
	lines := strings.Split(statusOut, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[2:])
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[len(parts)-1]
		}
		changes = append(changes, GitChange{
			Path:   path,
			Status: status,
		})
	}
	return changes, nil
}

func Show(path string, commit string) (string, error) {
	if commit == "" {
		commit = "HEAD"
	}
	return RunGitCommand("show", commit+":"+path)
}

func Blame(path string) ([]BlameLine, error) {
	blameOut, err := RunGitCommand("blame", "--porcelain", "--", path)
	if err != nil {
		return []BlameLine{}, err
	}

	var blameLines []BlameLine = []BlameLine{}
	lines := strings.Split(blameOut, "\n")
	
	commits := make(map[string]map[string]string)
	var currentCommit string
	var finalLineNum int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "\t") {
			attr := commits[currentCommit]
			blameLines = append(blameLines, BlameLine{
				Line:    finalLineNum,
				Commit:  currentCommit,
				Author:  attr["author"],
				Date:    attr["date"],
				Summary: attr["summary"],
			})
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		key := parts[0]
		val := parts[1]

		if len(key) == 40 {
			currentCommit = key
			headerParts := strings.Split(val, " ")
			if len(headerParts) >= 2 {
				fmt.Sscanf(headerParts[1], "%d", &finalLineNum)
			}
			if _, exists := commits[currentCommit]; !exists {
				commits[currentCommit] = make(map[string]string)
			}
			continue
		}

		if currentCommit != "" {
			if key == "author" {
				commits[currentCommit]["author"] = val
			} else if key == "author-time" {
				var sec int64
				fmt.Sscanf(val, "%d", &sec)
				t := time.Unix(sec, 0)
				commits[currentCommit]["date"] = t.Format("2006-01-02")
			} else if key == "summary" {
				commits[currentCommit]["summary"] = val
			}
		}
	}

	return blameLines, nil
}

func Log(path string) ([]CommitInfo, error) {
	formatStr := "%H|%an|%ae|%ad|%ar|%s"
	var logOut string
	var err error
	if path != "" {
		logOut, err = RunGitCommand("log", "-n", "50", "--follow", "--format="+formatStr, "--", path)
	} else {
		logOut, err = RunGitCommand("log", "-n", "50", "--format="+formatStr)
	}

	if err != nil {
		return []CommitInfo{}, err
	}

	commits := []CommitInfo{}
	lines := strings.Split(logOut, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}
		commits = append(commits, CommitInfo{
			Hash:     parts[0],
			Author:   parts[1],
			Email:    parts[2],
			Date:     parts[3],
			Relative: parts[4],
			Summary:  parts[5],
		})
	}

	return commits, nil
}

func GetCommitFiles(commit string) ([]GitChange, error) {
	out, err := RunGitCommand("diff-tree", "--no-commit-id", "--name-status", "-r", commit)
	if err != nil {
		return []GitChange{}, err
	}

	changes := []GitChange{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		changes = append(changes, GitChange{
			Path:   parts[1],
			Status: parts[0],
		})
	}

	return changes, nil
}

type CommitDetails struct {
	Hash     string      `json:"hash"`
	Author   string      `json:"author"`
	Email    string      `json:"email"`
	Date     string      `json:"date"`
	Relative string      `json:"relative"`
	Subject  string      `json:"subject"`
	Body     string      `json:"body"`
	Files    []GitChange `json:"files"`
}

func GetCommitDetails(hash string) (CommitDetails, error) {
	// 1. Get info formatted
	// Use %H (hash), %an (author name), %ae (author email), %ad (author date), %ar (relative date), %s (subject), %b (body)
	formatStr := "%H|%an|%ae|%ad|%ar|%s|%b"
	out, err := RunGitCommand("show", "-s", "--format="+formatStr, hash)
	if err != nil {
		return CommitDetails{}, err
	}

	parts := strings.SplitN(out, "|", 7)
	if len(parts) < 6 {
		return CommitDetails{}, fmt.Errorf("invalid commit info output: %s", out)
	}

	body := ""
	if len(parts) == 7 {
		body = parts[6]
	}

	// 2. Get files changed
	files, err := GetCommitFiles(hash)
	if err != nil {
		files = []GitChange{}
	}

	return CommitDetails{
		Hash:     parts[0],
		Author:   parts[1],
		Email:    parts[2],
		Date:     parts[3],
		Relative: parts[4],
		Subject:  parts[5],
		Body:     body,
		Files:    files,
	}, nil
}

type Stash struct {
	Index   int    `json:"index"`
	Branch  string `json:"branch"`
	Message string `json:"message"`
	Hash    string `json:"hash"`
}

type Tag struct {
	Name    string `json:"name"`
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

type Contributor struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Commits int    `json:"commits"`
}

func GetStashes() ([]Stash, error) {
	out, err := RunGitCommand("stash", "list", "--format=%gd|||%H|||%gs")
	if err != nil {
		return []Stash{}, err
	}

	stashes := []Stash{}
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|||", 3)
		if len(parts) < 3 {
			continue
		}
		// parts[0] is like "stash@{0}", extract index from it
		ref := parts[0] // e.g. "stash@{0}"
		idx := i
		// try to parse the index from the ref
		start := strings.Index(ref, "{")
		end := strings.Index(ref, "}")
		if start >= 0 && end > start {
			fmt.Sscanf(ref[start+1:end], "%d", &idx)
		}
		stashes = append(stashes, Stash{
			Index:   idx,
			Hash:    parts[1],
			Branch:  ref,
			Message: parts[2],
		})
	}
	return stashes, nil
}

func GetTags() ([]Tag, error) {
	out, err := RunGitCommand("tag", "-l", "--sort=-creatordate",
		"--format=%(refname:short)|||%(objectname:short)|||%(creatordate:short)|||%(subject)")
	if err != nil {
		return []Tag{}, err
	}

	tags := []Tag{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|||", 4)
		if len(parts) < 4 {
			continue
		}
		tags = append(tags, Tag{
			Name:    parts[0],
			Hash:    parts[1],
			Date:    parts[2],
			Message: parts[3],
		})
	}
	return tags, nil
}

func GetContributors() ([]Contributor, error) {
	out, err := RunGitCommand("shortlog", "-sne", "HEAD")
	if err != nil {
		return []Contributor{}, err
	}

	contributors := []Contributor{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// format: "  42\tName <email@example.com>"
		tabIdx := strings.Index(line, "\t")
		if tabIdx < 0 {
			continue
		}
		countStr := strings.TrimSpace(line[:tabIdx])
		rest := strings.TrimSpace(line[tabIdx+1:])

		var count int
		fmt.Sscanf(countStr, "%d", &count)

		name := rest
		email := ""
		ltIdx := strings.Index(rest, "<")
		gtIdx := strings.Index(rest, ">")
		if ltIdx >= 0 && gtIdx > ltIdx {
			name = strings.TrimSpace(rest[:ltIdx])
			email = rest[ltIdx+1 : gtIdx]
		}

		contributors = append(contributors, Contributor{
			Name:    name,
			Email:   email,
			Commits: count,
		})
	}
	return contributors, nil
}

func SearchCommits(q, author, file string) ([]CommitInfo, error) {
	args := []string{"log", "-n", "100", "--format=%H|%an|%ae|%ad|%ar|%s"}
	if q != "" {
		args = append(args, "--grep="+q, "-i")
	}
	if author != "" {
		args = append(args, "--author="+author)
	}
	if file != "" {
		args = append(args, "--", file)
	}

	out, err := RunGitCommand(args...)
	if err != nil {
		return []CommitInfo{}, err
	}

	commits := []CommitInfo{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}
		commits = append(commits, CommitInfo{
			Hash:     parts[0],
			Author:   parts[1],
			Email:    parts[2],
			Date:     parts[3],
			Relative: parts[4],
			Summary:  parts[5],
		})
	}
	return commits, nil
}

func GetLineHistory(path string, startLine, endLine int) ([]CommitInfo, error) {
	lArg := fmt.Sprintf("-L%d,%d:%s", startLine, endLine, path)
	out, err := RunGitCommand("log", "--no-patch", "--format=%H|%an|%ae|%ad|%ar|%s", lArg)
	if err != nil {
		return []CommitInfo{}, err
	}

	commits := []CommitInfo{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 6 {
			continue
		}
		commits = append(commits, CommitInfo{
			Hash:     parts[0],
			Author:   parts[1],
			Email:    parts[2],
			Date:     parts[3],
			Relative: parts[4],
			Summary:  parts[5],
		})
	}
	return commits, nil
}

// GraphCommit is a single commit with the parent links needed to lay out a
// visual commit graph on the frontend.
type GraphCommit struct {
	Hash    string   `json:"hash"`
	Parents []string `json:"parents"`
	Author  string   `json:"author"`
	Date    string   `json:"date"`
	Refs    string   `json:"refs"`
	Subject string   `json:"subject"`
}

// GetCommitGraphData returns structured commit data (across all refs, in
// topological/date order) for rendering a lane-based commit graph.
func GetCommitGraphData() ([]GraphCommit, error) {
	const sep = "\x1f"
	format := "%H" + sep + "%P" + sep + "%an" + sep + "%aI" + sep + "%D" + sep + "%s"
	out, err := RunGitCommand("log", "--all", "--date-order", "--pretty=format:"+format, "-n", "300")
	if err != nil {
		return nil, err
	}

	commits := []GraphCommit{}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, sep)
		if len(fields) < 6 {
			continue
		}
		parents := []string{}
		if p := strings.TrimSpace(fields[1]); p != "" {
			parents = strings.Fields(p)
		}
		commits = append(commits, GraphCommit{
			Hash:    fields[0],
			Parents: parents,
			Author:  fields[2],
			Date:    fields[3],
			Refs:    fields[4],
			Subject: fields[5],
		})
	}
	return commits, nil
}

func BlameAtCommit(path, commit string) ([]BlameLine, error) {
	blameOut, err := RunGitCommand("blame", "--porcelain", commit, "--", path)
	if err != nil {
		return []BlameLine{}, err
	}

	var blameLines []BlameLine = []BlameLine{}
	lines := strings.Split(blameOut, "\n")

	commits := make(map[string]map[string]string)
	var currentCommit string
	var finalLineNum int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "\t") {
			attr := commits[currentCommit]
			blameLines = append(blameLines, BlameLine{
				Line:    finalLineNum,
				Commit:  currentCommit,
				Author:  attr["author"],
				Date:    attr["date"],
				Summary: attr["summary"],
			})
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		key := parts[0]
		val := parts[1]

		if len(key) == 40 {
			currentCommit = key
			headerParts := strings.Split(val, " ")
			if len(headerParts) >= 2 {
				fmt.Sscanf(headerParts[1], "%d", &finalLineNum)
			}
			if _, exists := commits[currentCommit]; !exists {
				commits[currentCommit] = make(map[string]string)
			}
			continue
		}

		if currentCommit != "" {
			if key == "author" {
				commits[currentCommit]["author"] = val
			} else if key == "author-time" {
				var sec int64
				fmt.Sscanf(val, "%d", &sec)
				t := time.Unix(sec, 0)
				commits[currentCommit]["date"] = t.Format("2006-01-02")
			} else if key == "summary" {
				commits[currentCommit]["summary"] = val
			}
		}
	}

	return blameLines, nil
}

func CompareRefs(ref1, ref2 string) ([]GitChange, error) {
	out, err := RunGitCommand("diff", "--name-status", ref1+"..."+ref2)
	if err != nil {
		return []GitChange{}, err
	}

	changes := []GitChange{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		changes = append(changes, GitChange{
			Status: parts[0],
			Path:   parts[1],
		})
	}
	return changes, nil
}

func GetDiffStat(ref1, ref2 string) (string, error) {
	return RunGitCommand("diff", "--stat", ref1+"..."+ref2)
}


