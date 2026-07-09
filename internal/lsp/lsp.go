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

	"golang.org/x/net/websocket"
	"github.com/Ucok23/vidian/internal/config"
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
	Candidates []string // {ws} expands to the workspace dir; ~/ expands to $HOME
	Args       []string
	Npx        string // optional npm package for an `npx` fallback
	Install    string // human-facing install command, surfaced in the UI
}

var tsServer = serverDef{
	Candidates: []string{
		"{ws}/node_modules/.bin/typescript-language-server",
		"{ws}/frontend/node_modules/.bin/typescript-language-server",
		"typescript-language-server",
	},
	Args:    []string{"--stdio"},
	Npx:     "typescript-language-server",
	Install: "npm i -g typescript-language-server typescript",
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
	},
	"python": {
		Candidates: []string{"pylsp", "~/.local/bin/pylsp"},
		Install:    "pipx install 'python-lsp-server[all]'  (or: pip install python-lsp-server)",
	},
	"typescript": tsServer,
	"javascript": tsServer,
	"rust": {
		Candidates: []string{"rust-analyzer", "~/.local/bin/rust-analyzer"},
		Install:    "rustup component add rust-analyzer",
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

// resolveServer finds the launch command for lang within workspaceDir. It
// returns the resolved binary path, its arguments, the install hint, and
// whether a runnable server was found.
func resolveServer(lang, workspaceDir string) (path string, args []string, install string, ok bool) {
	def, exists := servers[lang]
	if !exists {
		return "", nil, "", false
	}
	for _, c := range def.Candidates {
		c = strings.ReplaceAll(c, "{ws}", workspaceDir)
		c = expandHome(c)
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
		Available bool   `json:"available"`
		Path      string `json:"path,omitempty"`
		Install   string `json:"install"`
	}
	out := make(map[string]status, len(servers))
	for lang := range servers {
		path, _, install, ok := resolveServer(lang, dir)
		out[lang] = status{Available: ok, Path: path, Install: install}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
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
