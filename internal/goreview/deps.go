// Episode 08 — the dependency review. Direct vs. indirect, replace/pseudo-
// version smells in go.mod, CGO entry point, removal-cost ratio, and
// govulncheck (reachability-based, run only if the binary is present — this
// package never installs tools on its own, mirroring the "graceful handling
// for missing language servers" convention used elsewhere in Vidian).
package goreview

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ModuleRef mirrors the fields of `go list -m -json` that this package uses.
type ModuleRef struct {
	Path     string `json:"Path"`
	Version  string `json:"Version"`
	Indirect bool   `json:"Indirect"`
	Main     bool   `json:"Main"`
}

func listModules(dir string) ([]ModuleRef, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list -m: %s", string(ee.Stderr))
		}
		return nil, fmt.Errorf("go list -m: %w", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var mods []ModuleRef
	for dec.More() {
		var m ModuleRef
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("decoding go list -m output: %w", err)
		}
		mods = append(mods, m)
	}
	return mods, nil
}

// DependencyRow is one direct dependency row of the episode-08 artifact.
type DependencyRow struct {
	Path             string  `json:"path"`
	Version          string  `json:"version"`
	Indirect         bool    `json:"indirect"`
	DirectlyImported bool    `json:"directlyImported"`
	RemovalCostFiles int     `json:"removalCostFiles"`
	RemovalCostRatio float64 `json:"removalCostRatio"`
}

// ModuleFinding is one go.mod / dependency-graph smell, cited exactly.
type ModuleFinding struct {
	Category string `json:"category"` // "indirect-mismatch" | "replace" | "pseudo-version" | "exclude"
	Module   string `json:"module"`
	Message  string `json:"message"`
	Detail   string `json:"detail"`
}

// DependencyReport is the full episode-08 artifact.
type DependencyReport struct {
	Modules              []DependencyRow `json:"modules"`
	Findings             []ModuleFinding `json:"findings"`
	CGOPackages          []string        `json:"cgoPackages"`
	GraphEdgeCount       int             `json:"graphEdgeCount"`
	GovulncheckAvailable bool            `json:"govulncheckAvailable"`
	VulnOutput           string          `json:"vulnOutput,omitempty"`
	VulnError            string          `json:"vulnError,omitempty"`
}

var pseudoVersionRe = regexp.MustCompile(`-\d{14}-[0-9a-f]{12}(\+incompatible)?$`)

var (
	replaceLineRe = regexp.MustCompile(`^replace\s+(\S+)`)
	excludeLineRe = regexp.MustCompile(`^exclude\s+(\S+)`)
)

// goModDirectives holds the parts of go.mod that `go list -m` doesn't
// surface directly: raw replace/exclude directives as written.
type goModDirectives struct {
	Replaces []string
	Excludes []string
}

func parseGoModDirectives(dir string) goModDirectives {
	var d goModDirectives
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return d
	}
	inReplace, inExclude := false, false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case line == "replace (":
			inReplace = true
		case line == "exclude (":
			inExclude = true
		case (inReplace || inExclude) && line == ")":
			inReplace, inExclude = false, false
		case inReplace:
			d.Replaces = append(d.Replaces, line)
		case inExclude:
			d.Excludes = append(d.Excludes, line)
		case replaceLineRe.MatchString(line):
			d.Replaces = append(d.Replaces, strings.TrimPrefix(line, "replace "))
		case excludeLineRe.MatchString(line):
			d.Excludes = append(d.Excludes, strings.TrimPrefix(line, "exclude "))
		}
	}
	return d
}

// runDependencyReview implements episode 08 against the already-listed
// internal packages (pkgs) and their import graph.
func runDependencyReview(dir string, mod ModuleInfo, pkgs []goListPackage) DependencyReport {
	report := DependencyReport{Modules: []DependencyRow{}, Findings: []ModuleFinding{}, CGOPackages: []string{}}

	mods, err := listModules(dir)
	if err != nil {
		report.Findings = append(report.Findings, ModuleFinding{
			Category: "error", Module: "", Message: "go list -m -json all failed", Detail: err.Error(),
		})
		return report
	}

	// Which module paths are actually imported by our own internal packages,
	// regardless of what go.mod's indirect tag claims.
	directlyImported := map[string]bool{}
	totalFiles := 0
	filesByModule := map[string]int{}
	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		totalFiles += len(p.GoFiles)
		seenForPkg := map[string]bool{}
		for _, imp := range p.Imports {
			for _, m := range mods {
				if m.Main {
					continue
				}
				if imp == m.Path || strings.HasPrefix(imp, m.Path+"/") {
					directlyImported[m.Path] = true
					if !seenForPkg[m.Path] {
						seenForPkg[m.Path] = true
						filesByModule[m.Path] += len(p.GoFiles)
					}
				}
			}
		}
	}

	for _, m := range mods {
		if m.Main {
			continue
		}
		if !m.Indirect && !directlyImported[m.Path] {
			// go.mod calls it direct but nothing in our source imports it —
			// unusual but not a finding worth surfacing on its own (could be
			// a tool/build dependency). Still listed in the table.
		}
		if m.Indirect && directlyImported[m.Path] {
			report.Findings = append(report.Findings, ModuleFinding{
				Category: "indirect-mismatch",
				Module:   m.Path,
				Message:  fmt.Sprintf("%q is marked indirect in go.mod but is directly imported by internal code.", m.Path),
				Detail:   "go.mod is likely stale — run `go mod tidy` to correct the indirect markers.",
			})
		}
		if pseudoVersionRe.MatchString(m.Version) {
			report.Findings = append(report.Findings, ModuleFinding{
				Category: "pseudo-version",
				Module:   m.Path,
				Message:  fmt.Sprintf("%q is pinned to a pseudo-version (%s) — no tagged release ever contained this commit.", m.Path, m.Version),
				Detail:   "Sometimes necessary, but worth a comment explaining why.",
			})
		}

		ratio := 0.0
		if totalFiles > 0 {
			ratio = float64(filesByModule[m.Path]) / float64(totalFiles)
		}
		report.Modules = append(report.Modules, DependencyRow{
			Path: m.Path, Version: m.Version, Indirect: m.Indirect,
			DirectlyImported: directlyImported[m.Path],
			RemovalCostFiles: filesByModule[m.Path], RemovalCostRatio: ratio,
		})
	}
	sort.Slice(report.Modules, func(i, j int) bool {
		if report.Modules[i].DirectlyImported != report.Modules[j].DirectlyImported {
			return report.Modules[i].DirectlyImported
		}
		return report.Modules[i].Path < report.Modules[j].Path
	})

	directives := parseGoModDirectives(dir)
	for _, r := range directives.Replaces {
		report.Findings = append(report.Findings, ModuleFinding{
			Category: "replace", Module: r,
			Message: fmt.Sprintf("replace directive: %s", r),
			Detail:  "Fine for a temporary patch or a fork; otherwise a supply-chain hole. Worth a comment explaining what the fork changes.",
		})
	}
	for _, e := range directives.Excludes {
		report.Findings = append(report.Findings, ModuleFinding{
			Category: "exclude", Module: e,
			Message: fmt.Sprintf("exclude directive: %s", e),
			Detail:  "Rare enough that it should have a story.",
		})
	}

	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Category != report.Findings[j].Category {
			return report.Findings[i].Category < report.Findings[j].Category
		}
		return report.Findings[i].Module < report.Findings[j].Module
	})

	// CGO: walk the full transitive closure (not just internal packages) —
	// this is the one place episode 08 needs external packages too.
	if cgoPkgs, edgeCount, err := findCGOEntry(dir); err == nil {
		report.CGOPackages = cgoPkgs
		report.GraphEdgeCount = edgeCount
	}

	// govulncheck: reachability-based, run only if already installed. This
	// tool never installs anything on the user's machine.
	if path, err := exec.LookPath("govulncheck"); err == nil {
		report.GovulncheckAvailable = true
		cmd := exec.Command(path, "./...")
		cmd.Dir = dir
		out, runErr := cmd.CombinedOutput()
		report.VulnOutput = string(out)
		if runErr != nil {
			if _, ok := runErr.(*exec.ExitError); !ok {
				report.VulnError = runErr.Error()
			}
		}
	}

	return report
}

type goListPackageWithCgo struct {
	ImportPath string   `json:"ImportPath"`
	CgoFiles   []string `json:"CgoFiles"`
}

// findCGOEntry walks the full transitive dependency closure (`go list -deps
// ./...`, including external and stdlib packages) looking for CgoFiles, and
// separately counts `go mod graph` edges as a blast-radius proxy.
func findCGOEntry(dir string) ([]string, int, error) {
	cmd := exec.Command("go", "list", "-json", "-deps", "./...")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, 0, err
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var cgoPkgs []string
	for dec.More() {
		var p goListPackageWithCgo
		if err := dec.Decode(&p); err != nil {
			break
		}
		if len(p.CgoFiles) > 0 {
			cgoPkgs = append(cgoPkgs, p.ImportPath)
		}
	}
	sort.Strings(cgoPkgs)

	edgeCount := 0
	graphCmd := exec.Command("go", "mod", "graph")
	graphCmd.Dir = dir
	if graphOut, err := graphCmd.Output(); err == nil {
		edgeCount = len(strings.Split(strings.TrimSpace(string(graphOut)), "\n"))
	}

	return cgoPkgs, edgeCount, nil
}
