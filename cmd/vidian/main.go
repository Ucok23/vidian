package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Ucok23/vidian"
	"github.com/Ucok23/vidian/internal/config"
	"github.com/Ucok23/vidian/internal/server"
)

func main() {
	var workspaceDir string
	var devMode bool
	var port int

	flag.StringVar(&workspaceDir, "dir", ".", "Directory to open as workspace")
	flag.BoolVar(&devMode, "dev", false, "Enable development mode (serve frontend from local disk)")
	flag.IntVar(&port, "port", 8080, "Port to run the server on")
	flag.Parse()

	// Resolve absolute path of workspace
	absPath, err := filepath.Abs(workspaceDir)
	if err != nil {
		log.Fatalf("Error resolving workspace directory path: %v", err)
	}
	workspaceDir = absPath

	// Check if directory exists
	info, err := os.Stat(workspaceDir)
	if err != nil {
		log.Fatalf("Workspace directory does not exist: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("Specified path is not a directory: %s", workspaceDir)
	}

	// If a Vidian instance is already serving this port, register the directory
	// with it and open a browser tab there instead of starting a second server.
	if id, ok := registerWithRunningInstance(port, workspaceDir); ok {
		url := fmt.Sprintf("http://localhost:%d/?ws=%s", port, id)
		log.Printf("Vidian is already running on port %d; opening %s", port, workspaceDir)
		server.OpenBrowser(url)
		return
	}

	log.Printf("Starting Vidian...")
	log.Printf("Workspace: %s", workspaceDir)
	log.Printf("Port: %d", port)
	log.Printf("Mode: %s", map[bool]string{true: "Development", false: "Embedded Production"}[devMode])

	cfg := config.New(devMode, port)
	ws := cfg.Add(workspaceDir)

	// Open the initial workspace in the browser once the server is listening.
	go func() {
		url := fmt.Sprintf("http://localhost:%d/?ws=%s", port, ws.ID)
		waitForServer(port)
		server.OpenBrowser(url)
	}()

	server.Start(cfg, vidian.EmbeddedFiles)
}

// registerWithRunningInstance checks whether a live Vidian instance is already
// serving the given port. If so, it POSTs the workspace path to it and returns
// the assigned workspace ID. Returns ok=false when no instance is reachable.
func registerWithRunningInstance(port int, absPath string) (string, bool) {
	client := &http.Client{Timeout: 500 * time.Millisecond}

	// Confirm the port belongs to a Vidian instance before posting to it.
	pingResp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/ping", port))
	if err != nil {
		return "", false
	}
	defer pingResp.Body.Close()
	var ping struct {
		App string `json:"app"`
	}
	if json.NewDecoder(pingResp.Body).Decode(&ping) != nil || ping.App != "vidian" {
		return "", false
	}

	body, _ := json.Marshal(map[string]string{"path": absPath})
	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/api/workspaces", port),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var ws config.Workspace
	if json.NewDecoder(resp.Body).Decode(&ws) != nil || ws.ID == "" {
		return "", false
	}
	return ws.ID, true
}

// waitForServer polls the local ping endpoint until the server responds or a
// short deadline passes, so the browser tab opens against a live listener.
func waitForServer(port int) {
	client := &http.Client{Timeout: 200 * time.Millisecond}
	url := fmt.Sprintf("http://localhost:%d/api/ping", port)
	for i := 0; i < 50; i++ {
		if resp, err := client.Get(url); err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
