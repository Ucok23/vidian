package lsp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
	"github.com/Ucok23/vidian/internal/config"
)

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

func HandleLSP(ws *websocket.Conn) {
	defer ws.Close()

	lang := ws.Request().URL.Query().Get("lang")
	log.Printf("LSP client connected over WebSocket. Language: %s", lang)

	var cmdPath string
	var cmdArgs []string

	switch lang {
	case "go":
		goplsPath, err := exec.LookPath("gopls")
		if err != nil {
			goplsPath = filepath.Join(os.Getenv("HOME"), "go", "bin", "gopls")
		}
		cmdPath = goplsPath
		cmdArgs = []string{"-mode", "stdio"}
	case "python":
		pylspPath, err := exec.LookPath("pylsp")
		if err != nil {
			pylspPath = filepath.Join(os.Getenv("HOME"), ".local/bin/pylsp")
		}
		cmdPath = pylspPath
		cmdArgs = []string{}
	case "typescript", "javascript":
		tsLspPath := filepath.Join(config.ActiveConfig.WorkspaceDir, "frontend", "node_modules", ".bin", "typescript-language-server")
		if _, err := os.Stat(tsLspPath); err != nil {
			cmdPath = "npx"
			cmdArgs = []string{"typescript-language-server", "--stdio"}
		} else {
			cmdPath = tsLspPath
			cmdArgs = []string{"--stdio"}
		}
	case "rust":
		rustLspPath, err := exec.LookPath("rust-analyzer")
		if err != nil {
			rustLspPath = filepath.Join(os.Getenv("HOME"), ".local/bin/rust-analyzer")
		}
		cmdPath = rustLspPath
		cmdArgs = []string{}
	default:
		log.Printf("Unsupported LSP language: %s", lang)
		sendLSPError(ws, fmt.Sprintf("No language server configured for %q.", lang))
		return
	}

	// For non-npx paths, verify the binary exists before trying to launch it.
	if cmdPath != "npx" {
		if _, err := exec.LookPath(cmdPath); err != nil {
			if _, err2 := os.Stat(cmdPath); err2 != nil {
				msg := fmt.Sprintf("Language server for %q not found (%s). Please install it and make sure it is on your PATH.", lang, cmdPath)
				log.Printf("LSP binary not found: %s", cmdPath)
				sendLSPError(ws, msg)
				return
			}
		}
	}

	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Dir = config.ActiveConfig.WorkspaceDir

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
