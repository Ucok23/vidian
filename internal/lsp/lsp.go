package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Ucok23/vidian/internal/config"
	"golang.org/x/net/websocket"
)

// serverDef declares how to launch a language server and how to install it.
//
// Candidates are tried in order. A candidate containing a path separator (or a
// leading ~/ or the {ws} placeholder) is treated as an explicit path and must
// exist on disk; a bare name is resolved against $PATH. If none resolve and Npx
// is set, `npx <Npx> <Args...>` is used as a last resort (npm auto-fetches the
// package on first run). This keeps every language's discovery, arguments, and
// install hint in one declarative place instead of a hand-written switch.
type serverDef struct {
	Candidates []string // {ws}→workspace dir, {vlsp}→Vidian LSP dir; ~/ expands to $HOME
	Args       []string
	Npx        string // optional npm package for an `npx` fallback
	Install    string // human-facing install command, surfaced in the UI

	// InstallRun is the argv Vidian runs when the user explicitly clicks
	// "Install". It is fixed server-side (the client only names a language) and
	// runs only on that explicit action. nil means "no safe auto-install" — the
	// UI falls back to showing Install for the user to run themselves. Installs
	// target user/Vidian-owned locations, never the workspace being viewed;
	// {vlsp} expands to the Vidian LSP dir for that.
	InstallRun []string
}

var tsServer = serverDef{
	Candidates: []string{
		"{ws}/node_modules/.bin/typescript-language-server",
		"{ws}/frontend/node_modules/.bin/typescript-language-server",
		"{vlsp}/ts/node_modules/.bin/typescript-language-server",
		"typescript-language-server",
	},
	Args:       []string{"--stdio"},
	Npx:        "typescript-language-server",
	Install:    "npm i -g typescript-language-server typescript",
	InstallRun: []string{"npm", "i", "--prefix", "{vlsp}/ts", "typescript-language-server", "typescript"},
}

var clangdServer = serverDef{
	Candidates: []string{"clangd", "~/.local/bin/clangd"},
	Install:    "Install clangd from your package manager (e.g. apt install clangd, brew install llvm)",
}

// servers maps a language id to its launch/install definition. Aliases (js↔ts,
// c↔cpp) share a def.
var servers = map[string]serverDef{
	"go": {
		Candidates: []string{"gopls", "~/go/bin/gopls"},
		Args:       []string{"-mode", "stdio"},
		Install:    "go install golang.org/x/tools/gopls@latest",
		InstallRun: []string{"go", "install", "golang.org/x/tools/gopls@latest"},
	},
	"python": {
		Candidates: []string{"pylsp", "~/.local/bin/pylsp"},
		Install:    "pipx install 'python-lsp-server[all]'  (or: pip install python-lsp-server)",
		InstallRun: []string{"pipx", "install", "python-lsp-server[all]"},
	},
	"typescript": tsServer,
	"javascript": tsServer,
	"rust": {
		Candidates: []string{"rust-analyzer", "~/.cargo/bin/rust-analyzer", "~/.local/bin/rust-analyzer"},
		Install:    "rustup component add rust-analyzer",
		InstallRun: []string{"rustup", "component", "add", "rust-analyzer"},
	},
	"c":   clangdServer,
	"cpp": clangdServer,
	"lua": {
		Candidates: []string{"lua-language-server", "~/.local/bin/lua-language-server"},
		Install:    "Download from https://github.com/LuaLS/lua-language-server/releases",
	},
	"ruby": {
		Candidates: []string{"solargraph", "~/.local/bin/solargraph"},
		Args:       []string{"stdio"},
		Install:    "gem install solargraph",
	},
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(os.Getenv("HOME"), p[2:])
	}
	return p
}

// vidianLspDir is where Vidian installs language servers it manages itself, so
// installs never touch the workspace being viewed. Honors XDG_DATA_HOME.
func vidianLspDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(base, "vidian", "lsp")
}

// expandPlaceholders resolves {ws}, {vlsp}, and a leading ~/ in a candidate or
// install-argv token.
func expandPlaceholders(p, workspaceDir string) string {
	p = strings.ReplaceAll(p, "{ws}", workspaceDir)
	p = strings.ReplaceAll(p, "{vlsp}", vidianLspDir())
	return expandHome(p)
}

// resolveServer finds the launch command for lang within workspaceDir. It
// returns the resolved binary path, its arguments, the install hint, and
// whether a runnable server was found.
func resolveServer(lang, workspaceDir string) (path string, args []string, install string, ok bool) {
	def, exists := servers[lang]
	if !exists {
		return "", nil, "", false
	}
	for _, c := range def.Candidates {
		c = expandPlaceholders(c, workspaceDir)
		if strings.ContainsRune(c, os.PathSeparator) {
			// Explicit path — must exist and be a file.
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
				return c, def.Args, def.Install, true
			}
			continue
		}
		// Bare name — resolve against $PATH.
		if lp, err := exec.LookPath(c); err == nil {
			return lp, def.Args, def.Install, true
		}
	}
	if def.Npx != "" {
		if lp, err := exec.LookPath("npx"); err == nil {
			return lp, append([]string{def.Npx}, def.Args...), def.Install, true
		}
	}
	return "", nil, def.Install, false
}

// isCleanClose reports whether err is an expected stream-close that happens on
// graceful shutdown or client disconnect (EOF, or a killed LSP process closing
// its pipes), as opposed to a real failure worth logging.
func isCleanClose(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed)
}

func sendLSPError(ws *websocket.Conn, msg string) {
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"window/showMessage","params":{"type":1,"message":%q}}`,
		msg,
	)
	framed := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	_ = websocket.Message.Send(ws, framed)
}

// sendServerUnavailable notifies the client that the requested language server
// is not installed, carrying the install hint so the UI can show it. This is a
// distinct method (not window/showMessage) so the client can treat it as an
// expected, actionable state rather than a generic error.
func sendServerUnavailable(ws *websocket.Conn, lang, install string) {
	params, _ := json.Marshal(map[string]string{"lang": lang, "install": install})
	body := fmt.Sprintf(`{"jsonrpc":"2.0","method":"vidian/serverUnavailable","params":%s}`, params)
	framed := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	_ = websocket.Message.Send(ws, framed)
}

// HandleLSPStatus reports, per configured language, whether a runnable server
// was found and how to install it otherwise. The client fetches this to avoid
// connecting to missing servers and to surface install hints in the UI.
func HandleLSPStatus(w http.ResponseWriter, r *http.Request) {
	dir := ""
	if wsCfg := config.ActiveConfig.Get(r.URL.Query().Get("ws")); wsCfg != nil {
		dir = wsCfg.Path
	}
	type status struct {
		Available  bool   `json:"available"`
		Path       string `json:"path,omitempty"`
		Install    string `json:"install"`
		CanInstall bool   `json:"canInstall"` // Vidian can run the install itself
	}
	out := make(map[string]status, len(servers))
	for lang, def := range servers {
		path, _, install, ok := resolveServer(lang, dir)
		out[lang] = status{Available: ok, Path: path, Install: install, CanInstall: len(def.InstallRun) > 0}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// flushWriter flushes the HTTP response after every write so install output
// streams to the browser line-by-line instead of buffering until the end.
type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

// installMu guards installing, which serializes concurrent install requests for
// the same language (a double-click shouldn't launch two npm installs).
var (
	installMu  sync.Mutex
	installing = map[string]bool{}
)

// HandleLSPInstall runs the fixed, per-language install command for the named
// language and streams its output back as plain text. The command is chosen
// entirely server-side from the servers table — the client only supplies a
// language id, never a command — and only runs on this explicit request. It
// targets user/Vidian-owned locations, never the workspace being viewed.
func HandleLSPInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dir := ""
	if wsCfg := config.ActiveConfig.Get(r.URL.Query().Get("ws")); wsCfg != nil {
		dir = wsCfg.Path
	}
	var body struct {
		Lang string `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	def, ok := servers[body.Lang]
	if !ok || len(def.InstallRun) == 0 {
		http.Error(w, "no automatic install available for this language", http.StatusBadRequest)
		return
	}

	// One install per language at a time.
	installMu.Lock()
	if installing[body.Lang] {
		installMu.Unlock()
		http.Error(w, "install already in progress", http.StatusConflict)
		return
	}
	installing[body.Lang] = true
	installMu.Unlock()
	defer func() {
		installMu.Lock()
		delete(installing, body.Lang)
		installMu.Unlock()
	}()

	// Resolve placeholders ({vlsp}, {ws}, ~/) into a concrete argv.
	argv := make([]string, len(def.InstallRun))
	for i, a := range def.InstallRun {
		argv[i] = expandPlaceholders(a, dir)
	}
	_ = os.MkdirAll(vidianLspDir(), 0o755) // for {vlsp}-prefixed installs

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	fw := &flushWriter{w: w, f: flusher}

	fmt.Fprintf(fw, "$ %s\n\n", strings.Join(argv, " "))

	// Fail cleanly if the installer tool itself is missing.
	if _, err := exec.LookPath(argv[0]); err != nil {
		fmt.Fprintf(fw, "Error: %q is not installed or not on your PATH.\nInstall it first, or run manually:\n  %s\n", argv[0], def.Install)
		return
	}

	cmd := exec.CommandContext(r.Context(), argv[0], argv[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = fw
	cmd.Stderr = fw
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(fw, "\nInstall failed: %v\n", err)
		return
	}

	if _, _, _, ok := resolveServer(body.Lang, dir); ok {
		fmt.Fprintf(fw, "\n%s language server installed ✓\n", body.Lang)
	} else {
		fmt.Fprintf(fw, "\nInstall finished, but the server still isn't detected.\nIt may need a new PATH entry — see the output above.\n")
	}
}

func HandleLSP(ws *websocket.Conn) {
	defer ws.Close()

	lang := ws.Request().URL.Query().Get("lang")
	log.Printf("LSP client connected over WebSocket. Language: %s", lang)

	// Resolve the workspace this LSP session is scoped to.
	workspace := config.ActiveConfig.Get(ws.Request().URL.Query().Get("ws"))
	if workspace == nil {
		log.Printf("LSP: unknown or missing workspace")
		sendLSPError(ws, "No workspace specified for language server.")
		return
	}
	workspaceDir := workspace.Path

	if _, known := servers[lang]; !known {
		log.Printf("Unsupported LSP language: %s", lang)
		sendLSPError(ws, fmt.Sprintf("No language server configured for %q.", lang))
		return
	}

	cmdPath, cmdArgs, install, ok := resolveServer(lang, workspaceDir)
	if !ok {
		// Missing binary is an expected, recoverable state — tell the client
		// specifically (with the install hint) so it can surface it and fall
		// back gracefully instead of failing silently.
		log.Printf("LSP server for %q not found", lang)
		sendServerUnavailable(ws, lang, install)
		return
	}

	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Dir = workspaceDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("LSP StdinPipe failed: %v", err)
		sendLSPError(ws, fmt.Sprintf("Failed to start language server for %q: %v", lang, err))
		return
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("LSP StdoutPipe failed: %v", err)
		sendLSPError(ws, fmt.Sprintf("Failed to start language server for %q: %v", lang, err))
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		log.Printf("LSP process start failed (%s): %v", cmdPath, err)
		sendLSPError(ws, fmt.Sprintf("Language server for %q failed to start: %v", lang, err))
		return
	}
	defer func() {
		log.Printf("Killing LSP process (%s)", cmdPath)
		cmd.Process.Kill()
		cmd.Wait()
	}()

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() {
		once.Do(func() {
			close(done)
		})
	}

	// Read stdout -> write to WebSocket
	go func() {
		defer closeDone()
		reader := bufio.NewReader(stdout)
		for {
			var contentLength int
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if !isCleanClose(err) {
						log.Printf("LSP reader error: %v", err)
					}
					return
				}
				line = strings.TrimSpace(line)
				if line == "" {
					break // end of headers
				}
				if strings.HasPrefix(line, "Content-Length:") {
					fmt.Sscanf(line, "Content-Length: %d", &contentLength)
				}
			}

			if contentLength <= 0 {
				continue
			}

			body := make([]byte, contentLength)
			_, err = io.ReadFull(reader, body)
			if err != nil {
				log.Printf("LSP read body failed: %v", err)
				return
			}

			err = websocket.Message.Send(ws, string(body))
			if err != nil {
				log.Printf("LSP ws send failed: %v", err)
				return
			}
		}
	}()

	// Read WebSocket -> write to stdin
	go func() {
		defer closeDone()
		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				if !isCleanClose(err) {
					log.Printf("LSP ws receive failed: %v", err)
				}
				return
			}

			framed := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(msg), msg)
			_, err = io.WriteString(stdin, framed)
			if err != nil {
				log.Printf("LSP stdin write failed: %v", err)
				return
			}
		}
	}()

	<-done
	log.Printf("LSP connection closed")
}
