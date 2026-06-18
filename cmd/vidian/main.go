package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

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

	log.Printf("Starting Vidian...")
	log.Printf("Workspace: %s", workspaceDir)
	log.Printf("Port: %d", port)
	log.Printf("Mode: %s", map[bool]string{true: "Development", false: "Embedded Production"}[devMode])

	cfg := &config.Config{
		WorkspaceDir: workspaceDir,
		DevMode:      devMode,
		Port:         port,
	}

	server.Start(cfg, vidian.EmbeddedFiles)
}
