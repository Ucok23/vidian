// Episode 06 — concurrency read as ownership. Deliberately narrow: only the
// patterns the course itself calls mechanical/greppable make it in here.
// Channel-close-ownership with multiple senders needs real data-flow
// tracing to call accurately (who sends, who's allowed to close) and is
// left out rather than risk a confidently-wrong finding — the one thing
// this whole tool is built to avoid.
package goreview

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// GoroutineFinding is one mechanical concurrency-ownership observation.
type GoroutineFinding struct {
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
	Detail  string `json:"detail"`
}

// ConcurrencyReport is the full episode-06 artifact.
type ConcurrencyReport struct {
	Findings []GoroutineFinding `json:"findings"`
	RaceInCI bool               `json:"raceInCI"`
}

type posRange struct{ start, end token.Pos }

func within(pos token.Pos, ranges []posRange) bool {
	for _, r := range ranges {
		if pos >= r.start && pos <= r.end {
			return true
		}
	}
	return false
}

func runConcurrencyReview(dir string, mod ModuleInfo, pkgs []goListPackage) ConcurrencyReport {
	report := ConcurrencyReport{Findings: []GoroutineFinding{}}
	fset := token.NewFileSet()

	for _, p := range pkgs {
		if !isInternal(mod, p.ImportPath) {
			continue
		}
		for _, gf := range p.GoFiles {
			path := filepath.Join(p.Dir, gf)
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				continue
			}
			rel, _ := filepath.Rel(dir, path)
			report.Findings = append(report.Findings, analyzeConcurrency(file, fset, rel)...)
		}
	}

	report.RaceInCI = checkRaceInCI(dir)
	return report
}

func analyzeConcurrency(file *ast.File, fset *token.FileSet, rel string) []GoroutineFinding {
	var findings []GoroutineFinding

	// Pass 1: collect the position span of every loop body, so later checks
	// can test "is this node inside a loop" by simple range containment.
	var loopRanges []posRange
	ast.Inspect(file, func(n ast.Node) bool {
		switch loop := n.(type) {
		case *ast.ForStmt:
			loopRanges = append(loopRanges, posRange{loop.Body.Pos(), loop.Body.End()})
		case *ast.RangeStmt:
			loopRanges = append(loopRanges, posRange{loop.Body.Pos(), loop.Body.End()})
		}
		return true
	})

	// Pass 2: goroutines started inside a loop, and time.After() inside a
	// select that's itself inside a loop.
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GoStmt:
			if within(node.Pos(), loopRanges) {
				pos := fset.Position(node.Pos())
				findings = append(findings, GoroutineFinding{
					Kind: "unbounded-goroutine-in-loop", File: rel, Line: pos.Line,
					Snippet: "go " + callExprString(node.Call),
					Detail:  "A goroutine started per loop iteration with no visible limiter. If the loop bound comes from external input, this is unbounded concurrency — use errgroup.SetLimit or a fixed-size worker pool.",
				})
			}
		case *ast.SelectStmt:
			if !within(node.Pos(), loopRanges) {
				return true
			}
			for _, clause := range node.Body.List {
				cc, ok := clause.(*ast.CommClause)
				if !ok {
					continue
				}
				ast.Inspect(cc, func(inner ast.Node) bool {
					if call, ok := inner.(*ast.CallExpr); ok && isSelectorCall(call, "time", "After") {
						pos := fset.Position(call.Pos())
						findings = append(findings, GoroutineFinding{
							Kind: "time-after-in-loop-select", File: rel, Line: pos.Line,
							Snippet: "time.After(...)",
							Detail:  "time.After inside a select in a loop allocates a new timer every iteration; the old one isn't collected until it fires. Use time.NewTimer + Stop, or context.WithTimeout.",
						})
					}
					return true
				})
			}
		}
		return true
	})

	// Pass 3: errgroup.WithContext(ctx) whose derived ctx is discarded.
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || !isSelectorCall(call, "errgroup", "WithContext") {
			return true
		}
		if ident, ok := assign.Lhs[1].(*ast.Ident); ok && ident.Name == "_" {
			pos := fset.Position(assign.Pos())
			findings = append(findings, GoroutineFinding{
				Kind: "errgroup-discarded-ctx", File: rel, Line: pos.Line,
				Snippet: "_, _ = errgroup.WithContext(ctx)",
				Detail:  "The derived ctx from errgroup.WithContext is discarded. Cancellation becomes decorative: the first goroutine's error is returned to the caller while the others keep running against a request nobody is waiting for.",
			})
		}
		return true
	})

	// Pass 4 (per top-level function): time.NewTicker without a matching
	// .Stop() call anywhere in the same function (including deferred
	// closures — inspecting the whole FuncDecl body covers that).
	ast.Inspect(file, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			return true
		}
		tickerVars := map[string]token.Pos{}
		stopped := map[string]bool{}
		ast.Inspect(fd.Body, func(inner ast.Node) bool {
			if assign, ok := inner.(*ast.AssignStmt); ok && len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
				if call, ok := assign.Rhs[0].(*ast.CallExpr); ok && isSelectorCall(call, "time", "NewTicker") {
					if ident, ok := assign.Lhs[0].(*ast.Ident); ok {
						tickerVars[ident.Name] = assign.Pos()
					}
				}
			}
			if call, ok := inner.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Stop" {
					if ident, ok := sel.X.(*ast.Ident); ok {
						stopped[ident.Name] = true
					}
				}
			}
			return true
		})
		for name, pos := range tickerVars {
			if !stopped[name] {
				p := fset.Position(pos)
				findings = append(findings, GoroutineFinding{
					Kind: "ticker-without-stop", File: rel, Line: p.Line,
					Snippet: name + " := time.NewTicker(...)",
					Detail:  "No matching " + name + ".Stop() found in this function — the ticker's goroutine leaks for the life of the program.",
				})
			}
		}
		return false // don't also descend via the outer Inspect into nested FuncLits as separate scopes
	})

	return findings
}

// isSelectorCall reports whether call is `pkg.Fn(...)` where pkg is an
// identifier whose name matches pkgHint (by exact match or import alias
// convention — good enough since it's only used to recognize a handful of
// well-known stdlib/x/sync APIs, not to resolve arbitrary types).
func isSelectorCall(call *ast.CallExpr, pkgHint, fn string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != fn {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == pkgHint
}

func callExprString(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name + "(...)"
	}
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return ident.Name + "(...)"
	}
	return "func(...)"
}

func checkRaceInCI(dir string) bool {
	found := false
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
		if strings.Contains(text, "go test") && strings.Contains(text, "-race") {
			found = true
		}
		return nil
	})
	return found
}
