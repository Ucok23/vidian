// Package goreview implements the deterministic, mechanical checks from the
// "Reviewing a Go Codebase End to End" course — episodes 02 (repo map), 03
// (package boundaries), 04 (interfaces at the wrong end) and 07 (context
// propagation). Every check here is pure computation over go.mod, `go list`,
// and the AST: no judgment calls, no LLM. The rule from the course carries
// over directly: flag and cite the exact location, never grade.
package goreview

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Status is a cheap check for whether a workspace is a Go module at all,
// used to decide whether the Go Review sidebar tab should appear.
type Status struct {
	IsGoModule bool   `json:"isGoModule"`
	ModulePath string `json:"modulePath,omitempty"`
	GoVersion  string `json:"goVersion,omitempty"`
}

// CheckStatus reports whether dir contains a go.mod, without running any
// analysis.
func CheckStatus(dir string) Status {
	mod, err := parseGoMod(dir)
	if err != nil {
		return Status{IsGoModule: false}
	}
	return Status{IsGoModule: true, ModulePath: mod.ModulePath, GoVersion: mod.GoVersion}
}

// ModuleInfo is the parsed subset of go.mod that episode 02 reads.
type ModuleInfo struct {
	ModulePath         string `json:"modulePath"`
	GoVersion          string `json:"goVersion"`
	DirectRequireCount int    `json:"directRequireCount"`
}

var (
	moduleRe = regexp.MustCompile(`^module\s+(\S+)`)
	goVerRe  = regexp.MustCompile(`^go\s+(\S+)`)
	// requireLineRe matches a single-line `require foo/bar v1.2.3` or a line
	// inside a `require (...)` block. Indirect deps carry a trailing
	// "// indirect" comment, which is how direct vs. indirect is told apart.
	requireLineRe = regexp.MustCompile(`^\s*([^\s/][^\s]*)\s+v[^\s]+`)
)

func parseGoMod(dir string) (ModuleInfo, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ModuleInfo{}, err
	}
	var mod ModuleInfo
	inRequireBlock := false
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		switch {
		case moduleRe.MatchString(trimmed):
			mod.ModulePath = moduleRe.FindStringSubmatch(trimmed)[1]
		case goVerRe.MatchString(trimmed):
			mod.GoVersion = goVerRe.FindStringSubmatch(trimmed)[1]
		case trimmed == "require (":
			inRequireBlock = true
		case inRequireBlock && trimmed == ")":
			inRequireBlock = false
		case inRequireBlock || strings.HasPrefix(trimmed, "require "):
			l := strings.TrimPrefix(trimmed, "require ")
			if requireLineRe.MatchString(l) && !strings.Contains(l, "// indirect") {
				mod.DirectRequireCount++
			}
		}
	}
	if mod.ModulePath == "" {
		return mod, fmt.Errorf("no module directive found")
	}
	return mod, nil
}

// goListPackage mirrors the fields of `go list -json` that this package uses.
type goListPackage struct {
	Dir        string   `json:"Dir"`
	ImportPath string   `json:"ImportPath"`
	Name       string   `json:"Name"`
	Imports    []string `json:"Imports"`
	Deps       []string `json:"Deps"`
	GoFiles    []string `json:"GoFiles"`
}

// listPackages runs `go list -json ./...` in dir and decodes the
// newline-concatenated JSON stream it produces (one object per package, not
// a JSON array).
//
// Note a gap this leaves: a directory whose build constraints exclude every
// file in it (a cgo-archive-only main, a platform-gated main, etc.) is
// dropped from `./...` entirely — `go list` exits 0 with no error and just
// never mentions it, with or without -e. buildRepoMap compensates by
// scanning the file tree directly for a `package main` in any directory
// that pkgs doesn't cover, since that's the only way to find it.
func listPackages(dir string) ([]goListPackage, error) {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list: %s", string(ee.Stderr))
		}
		return nil, fmt.Errorf("go list: %w", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var pkgs []goListPackage
	for dec.More() {
		var p goListPackage
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decoding go list output: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// PackageInfo is one internal package in the module's import graph.
type PackageInfo struct {
	ImportPath string   `json:"importPath"`
	Name       string   `json:"name"`
	Dir        string   `json:"dir,omitempty"`
	Imports    []string `json:"imports"`
	FanIn      int      `json:"fanIn"`
	FanOut     int      `json:"fanOut"`
}

// HotFile is one of the largest non-test source files.
type HotFile struct {
	Path  string `json:"path"`
	Lines int    `json:"lines"`
}

// Entrypoint is a `main` package discovered under the module.
type Entrypoint struct {
	Dir           string `json:"dir"`
	BuildTagGated bool   `json:"buildTagGated,omitempty"`
	Note          string `json:"note,omitempty"`
}

// RepoMap is the episode-02 artifact: one page, five sections.
type RepoMap struct {
	Module       ModuleInfo    `json:"module"`
	Entrypoints  []Entrypoint  `json:"entrypoints"`
	Layers       []PackageInfo `json:"layers"`
	HotFiles     []HotFile     `json:"hotFiles"`
	TestLines    int           `json:"testLines"`
	NonTestLines int           `json:"nonTestLines"`
	Unknowns     []string      `json:"unknowns"`
}

// Finding is a single cited, mechanical observation (episode 03: package
// boundaries). Never a verdict — always a location plus what to check.
type Finding struct {
	Category string `json:"category"` // "landfill" | "inverted-edge"
	Package  string `json:"package"`
	Message  string `json:"message"`
	Detail   string `json:"detail"`
}

// InterfaceRow is one row of the episode-04 artifact: interface, declaring
// package, implementations, consumers.
type InterfaceRow struct {
	Name            string   `json:"name"`
	Package         string   `json:"package"`
	File            string   `json:"file"`
	Line            int      `json:"line"`
	MethodCount     int      `json:"methodCount"`
	Implementations []string `json:"implementations"`
	ConsumerPkgs    []string `json:"consumerPkgs"`
	Flagged         bool     `json:"flagged"`
	Reason          string   `json:"reason,omitempty"`
}

// ContextFinding is one episode-07 grep hit.
type ContextFinding struct {
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

// Report is the full result of one Go Review run.
type Report struct {
	Module          ModuleInfo        `json:"module"`
	RepoMap         RepoMap           `json:"repoMap"`
	PackageFindings []Finding         `json:"packageFindings"`
	Interfaces      []InterfaceRow    `json:"interfaces"`
	ContextFindings []ContextFinding  `json:"contextFindings"`
	Dependencies    DependencyReport  `json:"dependencies"`
	Build           BuildReport       `json:"build"`
	Lint            LintReport        `json:"lint"`
	Concurrency     ConcurrencyReport `json:"concurrency"`
}

var landfillNames = map[string]bool{
	"utils": true, "util": true, "common": true, "helpers": true,
	"helper": true, "shared": true, "lib": true,
}

var domainPathHint = regexp.MustCompile(`(^|/)(domain|model|models|entity|entities)(/|$)`)

var invertedImportHints = []string{
	"net/http", "database/sql", "modernc.org/sqlite", "github.com/mattn/go-sqlite3",
	"gorm.io/gorm", "github.com/gin-gonic/gin", "github.com/go-chi/chi",
}

// Run executes the full deterministic Go Review (episodes 02/03/04/07)
// against dir, which must be the root of a Go module.
func Run(dir string) (*Report, error) {
	mod, err := parseGoMod(dir)
	if err != nil {
		return nil, err
	}

	pkgs, err := listPackages(dir)
	if err != nil {
		return nil, err
	}

	report := &Report{Module: mod}

	// The AST-only passes are fast (in-process parsing, no subprocesses) and
	// run up front. Dependencies/Build/Lint each shell out to `go build`,
	// `go vet`, golangci-lint, govulncheck, etc. — each one independently
	// slow on a real codebase — so they run concurrently rather than back
	// to back; wall-clock time drops to roughly the slowest of the three
	// instead of their sum.
	report.RepoMap = buildRepoMap(dir, mod, pkgs)
	report.PackageFindings = findPackageBoundaryIssues(mod, pkgs)
	report.Interfaces = findInterfacesAtWrongEnd(dir, mod, pkgs)
	report.ContextFindings = findContextIssues(dir, mod, pkgs)
	report.Concurrency = runConcurrencyReview(dir, mod, pkgs)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		report.Dependencies = runDependencyReview(dir, mod, pkgs)
	}()
	go func() {
		defer wg.Done()
		report.Build = runBuildReview(dir, mod)
	}()
	go func() {
		defer wg.Done()
		report.Lint = runLintReview(dir, mod)
	}()
	wg.Wait()

	return report, nil
}

func isInternal(mod ModuleInfo, importPath string) bool {
	return importPath == mod.ModulePath || strings.HasPrefix(importPath, mod.ModulePath+"/")
}

// buildRepoMap produces the episode-02 artifact.
func buildRepoMap(dir string, mod ModuleInfo, pkgs []goListPackage) RepoMap {
	rm := RepoMap{Module: mod, Unknowns: []string{}}

	// fan-in / fan-out over the internal import graph.
	fanIn := map[string]int{}
	byPath := map[string]goListPackage{}
	for _, p := range pkgs {
		byPath[p.ImportPath] = p
	}
	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		for _, imp := range p.Imports {
			if isInternal(mod, imp) && imp != p.ImportPath {
				fanIn[imp]++
			}
		}
	}

	var layers []PackageInfo
	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		if p.Name == "main" {
			rel, _ := filepath.Rel(dir, p.Dir)
			rm.Entrypoints = append(rm.Entrypoints, Entrypoint{Dir: rel})
		}
		fanOut := 0
		for _, imp := range p.Imports {
			if isInternal(mod, imp) && imp != p.ImportPath {
				fanOut++
			}
		}
		layers = append(layers, PackageInfo{
			ImportPath: p.ImportPath,
			Name:       p.Name,
			Imports:    p.Imports,
			FanIn:      fanIn[p.ImportPath],
			FanOut:     fanOut,
		})
	}
	// Layer order: packages with fewer internal dependencies first (a
	// topological-order proxy — true topo sort isn't needed to read the
	// shape of the graph).
	sort.SliceStable(layers, func(i, j int) bool {
		if layers[i].FanOut != layers[j].FanOut {
			return layers[i].FanOut < layers[j].FanOut
		}
		return layers[i].ImportPath < layers[j].ImportPath
	})
	rm.Layers = layers

	// go list ./... silently drops any directory where build constraints
	// exclude every file in it (a cgo-archive-only or platform-gated main,
	// for example) — it exits 0 with no error, so pkgs never mentions that
	// directory at all, not even with -e. The only way to find a `main`
	// hiding behind a build tag is to check directories go list didn't
	// cover, which the hot-files walk below is already visiting anyway.
	knownDirs := map[string]bool{}
	for _, p := range pkgs {
		knownDirs[p.Dir] = true
	}
	gatedMainDirs := map[string]string{} // dir -> file that declares package main

	// Hot files + test ratio: walk the tree once.
	var hot []HotFile
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" || name == "dist" || name == "testdata" ||
				strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		isTest := strings.HasSuffix(path, "_test.go")
		pdir := filepath.Dir(path)
		if !isTest && !knownDirs[pdir] {
			if _, already := gatedMainDirs[pdir]; !already {
				if data, err := os.ReadFile(path); err == nil && packageMainRe.Match(data) {
					gatedMainDirs[pdir] = filepath.Base(path)
				}
			}
		}
		lines := countLines(path)
		rel, _ := filepath.Rel(dir, path)
		if isTest {
			rm.TestLines += lines
		} else {
			rm.NonTestLines += lines
			hot = append(hot, HotFile{Path: rel, Lines: lines})
		}
		return nil
	})
	sort.Slice(hot, func(i, j int) bool { return hot[i].Lines > hot[j].Lines })
	if len(hot) > 5 {
		hot = hot[:5]
	}
	rm.HotFiles = hot

	for pdir, file := range gatedMainDirs {
		rel, _ := filepath.Rel(dir, pdir)
		rm.Entrypoints = append(rm.Entrypoints, Entrypoint{
			Dir:           rel,
			BuildTagGated: true,
			Note:          "package main in " + file + " is excluded by its build constraints under the default build — not visible to a plain `go build ./...`.",
		})
	}
	sort.Slice(rm.Entrypoints, func(i, j int) bool { return rm.Entrypoints[i].Dir < rm.Entrypoints[j].Dir })

	if len(rm.Entrypoints) == 0 {
		rm.Unknowns = append(rm.Unknowns, "No `main` package found under this module — entrypoint unknown.")
	}
	if rm.TestLines == 0 {
		rm.Unknowns = append(rm.Unknowns, "No _test.go files found anywhere in the module.")
	}

	return rm
}

var packageMainRe = regexp.MustCompile(`(?m)^package\s+main\s*$`)

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		n++
	}
	return n
}

// findPackageBoundaryIssues produces the episode-03 artifact: landfill
// candidates and inverted edges. Every finding cites the exact edge.
func findPackageBoundaryIssues(mod ModuleInfo, pkgs []goListPackage) []Finding {
	fanIn := map[string]int{}
	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		for _, imp := range p.Imports {
			if isInternal(mod, imp) && imp != p.ImportPath {
				fanIn[imp]++
			}
		}
	}

	findings := []Finding{}
	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) || p.Name == "main" {
			continue
		}
		fanOut := 0
		for _, imp := range p.Imports {
			if isInternal(mod, imp) && imp != p.ImportPath {
				fanOut++
			}
		}
		base := filepath.Base(p.ImportPath)

		if landfillNames[base] && fanIn[p.ImportPath] >= 2 && fanOut == 0 {
			findings = append(findings, Finding{
				Category: "landfill",
				Package:  p.ImportPath,
				Message:  fmt.Sprintf("Package %q is imported by %d internal package(s) and imports none itself.", p.ImportPath, fanIn[p.ImportPath]),
				Detail:   "Candidate landfill, not a proven one — check whether it's a vocabulary (has invariants/behavior) or a bag of unrelated helpers.",
			})
		}

		if domainPathHint.MatchString(p.ImportPath) {
			for _, imp := range p.Imports {
				for _, hint := range invertedImportHints {
					if imp == hint || strings.HasPrefix(imp, hint) {
						findings = append(findings, Finding{
							Category: "inverted-edge",
							Package:  p.ImportPath,
							Message:  fmt.Sprintf("%q imports %q.", p.ImportPath, imp),
							Detail:   "A domain-shaped package importing an HTTP/DB concern is usually an inverted dependency — check whether this edge can be justified in one sentence.",
						})
					}
				}
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Category != findings[j].Category {
			return findings[i].Category < findings[j].Category
		}
		return findings[i].Package < findings[j].Package
	})
	return findings
}

// methodSet is the set of method names declared with a given receiver type,
// keyed as "pkgImportPath.TypeName".
type typeKey struct {
	pkg  string
	name string
}

// findInterfacesAtWrongEnd implements episode 04's mechanical detection:
// for each interface, count implementations (types whose method set is a
// superset of the interface's method names) and flag the signature defect —
// exactly one implementation, declared in the same package as the interface.
//
// Matching is by method name only, not full signature equality. That is a
// deliberate simplification (avoids a full go/types pass) and is noted in
// the Reason field of ambiguous rows rather than hidden.
func findInterfacesAtWrongEnd(dir string, mod ModuleInfo, pkgs []goListPackage) []InterfaceRow {
	fset := token.NewFileSet()

	type ifaceDecl struct {
		row     InterfaceRow
		methods map[string]bool
	}
	var ifaces []*ifaceDecl
	methodSets := map[typeKey]map[string]bool{}
	// consumers: interface name -> set of packages (other than its own) that
	// reference that identifier as a type.
	consumerRefs := map[string]map[string]bool{}

	pkgByDir := map[string]goListPackage{}
	for _, p := range pkgs {
		pkgByDir[p.Dir] = p
	}

	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		for _, gf := range p.GoFiles {
			path := filepath.Join(p.Dir, gf)
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				continue
			}
			rel, _ := filepath.Rel(dir, path)

			ast.Inspect(file, func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						it, ok := ts.Type.(*ast.InterfaceType)
						if !ok {
							continue
						}
						methods := map[string]bool{}
						for _, m := range it.Methods.List {
							for _, name := range m.Names {
								methods[name.Name] = true
							}
						}
						if len(methods) == 0 {
							continue // empty interface{} — nothing to match against
						}
						pos := fset.Position(ts.Pos())
						ifaces = append(ifaces, &ifaceDecl{
							row: InterfaceRow{
								Name:        ts.Name.Name,
								Package:     p.ImportPath,
								File:        rel,
								Line:        pos.Line,
								MethodCount: len(methods),
							},
							methods: methods,
						})
					}
				case *ast.FuncDecl:
					if decl.Recv != nil && len(decl.Recv.List) > 0 {
						tn := receiverTypeName(decl.Recv.List[0].Type)
						if tn != "" {
							key := typeKey{pkg: p.ImportPath, name: tn}
							if methodSets[key] == nil {
								methodSets[key] = map[string]bool{}
							}
							methodSets[key][decl.Name.Name] = true
						}
					}
				}
				// Track identifiers used as parameter/field types, to
				// approximate "who consumes this interface".
				if ident, ok := n.(*ast.Ident); ok {
					if consumerRefs[ident.Name] == nil {
						consumerRefs[ident.Name] = map[string]bool{}
					}
					consumerRefs[ident.Name][p.ImportPath] = true
				}
				return true
			})
		}
	}

	rows := []InterfaceRow{}
	for _, decl := range ifaces {
		var impls []string
		for key, methods := range methodSets {
			if isSuperset(methods, decl.methods) {
				impls = append(impls, key.pkg+"."+key.name)
			}
		}
		sort.Strings(impls)

		consumers := []string{}
		for pkg := range consumerRefs[decl.row.Name] {
			if pkg != decl.row.Package {
				consumers = append(consumers, pkg)
			}
		}
		sort.Strings(consumers)

		row := decl.row
		row.Implementations = impls
		row.ConsumerPkgs = consumers

		sameFilePkgImpl := false
		for _, impl := range impls {
			if strings.HasPrefix(impl, decl.row.Package+".") {
				sameFilePkgImpl = true
			}
		}
		if len(impls) == 1 && sameFilePkgImpl {
			row.Flagged = true
			row.Reason = "One implementation, declared in the same package as the interface — the episode-04 signature defect. Check whether the consumer needs the full method set."
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Flagged != rows[j].Flagged {
			return rows[i].Flagged // flagged first
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func isSuperset(set, subset map[string]bool) bool {
	for m := range subset {
		if !set[m] {
			return false
		}
	}
	return true
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr: // generic receiver T[P]
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return ""
}

var (
	ctxBackgroundRe = regexp.MustCompile(`context\.(Background|TODO)\(\)`)
	httpDefaultRe   = regexp.MustCompile(`http\.(DefaultClient|Get|Post|PostForm)\(`)
)

// findContextIssues implements episode 07's greps plus two AST checks
// (context in a struct field, ctx not the first parameter).
func findContextIssues(dir string, mod ModuleInfo, pkgs []goListPackage) []ContextFinding {
	fset := token.NewFileSet()
	findings := []ContextFinding{}

	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		for _, gf := range p.GoFiles {
			path := filepath.Join(p.Dir, gf)
			rel, _ := filepath.Rel(dir, path)
			base := filepath.Base(path)
			isMain := base == "main.go"

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			text := string(data)

			if !isMain {
				for _, m := range findLineMatches(text, ctxBackgroundRe) {
					kind := "context.Background()"
					if strings.Contains(m.snippet, "TODO") {
						kind = "context.TODO()"
					}
					findings = append(findings, ContextFinding{
						Kind: kind, File: rel, Line: m.line, Snippet: m.snippet,
					})
				}
			}
			for _, m := range findLineMatches(text, httpDefaultRe) {
				findings = append(findings, ContextFinding{
					Kind: "http call with no explicit timeout/context", File: rel, Line: m.line, Snippet: m.snippet,
				})
			}

			file, err := parser.ParseFile(fset, path, data, 0)
			if err != nil {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.StructType:
					if decl.Fields == nil {
						return true
					}
					for _, f := range decl.Fields.List {
						if isContextType(f.Type) {
							pos := fset.Position(f.Pos())
							findings = append(findings, ContextFinding{
								Kind: "context.Context stored in a struct field", File: rel, Line: pos.Line,
								Snippet: exprString(f),
							})
						}
					}
				case *ast.FuncDecl:
					params := decl.Type.Params
					if params == nil {
						return true
					}
					idx := 0
					for _, f := range params.List {
						isCtx := isContextType(f.Type)
						for range f.Names {
							if isCtx && idx != 0 {
								pos := fset.Position(f.Pos())
								findings = append(findings, ContextFinding{
									Kind: "ctx is not the first parameter", File: rel, Line: pos.Line,
									Snippet: decl.Name.Name + "(...)",
								})
							}
							idx++
						}
						if len(f.Names) == 0 {
							idx++
						}
					}
				}
				return true
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Line < findings[j].Line
	})
	return findings
}

func isContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "context" && sel.Sel.Name == "Context"
}

func exprString(f *ast.Field) string {
	names := []string{}
	for _, n := range f.Names {
		names = append(names, n.Name)
	}
	if len(names) == 0 {
		return "context.Context"
	}
	return strings.Join(names, ", ") + " context.Context"
}

type lineMatch struct {
	line    int
	snippet string
}

func findLineMatches(text string, re *regexp.Regexp) []lineMatch {
	var matches []lineMatch
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		loc := re.FindStringIndex(line)
		if loc == nil {
			continue
		}
		// Skip matches that are themselves inside a quoted string literal
		// (e.g. this analyzer's own source printing the pattern it looks
		// for) rather than an actual call — checked by seeing whether the
		// match is immediately wrapped in `"`.
		if loc[0] > 0 && line[loc[0]-1] == '"' && loc[1] < len(line) && line[loc[1]] == '"' {
			continue
		}
		matches = append(matches, lineMatch{line: i + 1, snippet: trimmed})
	}
	return matches
}
