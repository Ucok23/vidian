package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/net/websocket"
)

//go:embed all:frontend/dist
var embeddedFiles embed.FS

type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"` // relative to workspace root
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

type WorkspaceInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type SearchResult struct {
	Path        string `json:"path"`
	LineNumber  int    `json:"lineNumber"`
	LineContent string `json:"lineContent"`
}

var (
	workspaceDir string
	devMode      bool
	port         int
)

func main() {
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

	// API Handlers
	http.HandleFunc("/api/workspace", handleWorkspace)
	http.HandleFunc("/api/dir", handleDir)
	http.HandleFunc("/api/file", handleFile)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/files", handleFiles)
	http.HandleFunc("/api/git/branches", handleGitBranches)
	http.HandleFunc("/api/git/checkout", handleGitCheckout)
	http.Handle("/api/lsp", websocket.Handler(handleLSP))

	// Frontend Handlers
	if devMode {
		// Serve from frontend/dist
		frontendDist := "./frontend/dist"
		log.Printf("Serving static assets from local path: %s", frontendDist)
		fileServer := http.FileServer(http.Dir(frontendDist))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Serve static files or fallback to index.html for SPA routing
			path := filepath.Join(frontendDist, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(frontendDist, "index.html"))
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	} else {
		// Serve from embed.FS
		log.Printf("Serving embedded static assets")
		distFS, err := fs.Sub(embeddedFiles, "frontend/dist")
		if err != nil {
			log.Fatalf("Error accessing embedded assets: %v", err)
		}
		fileServer := http.FileServer(http.FS(distFS))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Check if file exists in embedded FS
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			_, err := distFS.Open(path)
			if err != nil {
				// Fallback to index.html
				indexFile, err := distFS.Open("index.html")
				if err != nil {
					http.Error(w, "Index file not found in embed", http.StatusNotFound)
					return
				}
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				io.Copy(w, indexFile)
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	log.Printf("Server listening on http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// Helper to validate and clean input paths to prevent directory traversal
func getSafePath(reqPath string) (string, error) {
	// Join and clean path
	targetPath := filepath.Clean(filepath.Join(workspaceDir, reqPath))
	// Ensure the clean target path has the workspace directory as a prefix
	rel, err := filepath.Rel(workspaceDir, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("access denied: path lies outside workspace root")
	}
	return targetPath, nil
}

func setupCORS(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleWorkspace(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	info := WorkspaceInfo{
		Name: filepath.Base(workspaceDir),
		Path: workspaceDir,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleDir(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	relPath := r.URL.Query().Get("path")
	safePath, err := getSafePath(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read directory: %v", err), http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		name := entry.Name()
		// Skip dotfiles/folders like .git, except for configurations like .github or .vscode if wanted
		if name == ".git" || name == "node_modules" || name == ".DS_Store" {
			continue
		}

		info, err := entry.Info()
		var size int64
		if err == nil {
			size = info.Size()
		}

		entryRelPath := filepath.Join(relPath, name)
		// Clean windows backslashes to forward slashes for URLs
		entryRelPath = filepath.ToSlash(entryRelPath)

		files = append(files, FileInfo{
			Name:  name,
			Path:  entryRelPath,
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func handleFile(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	relPath := r.URL.Query().Get("path")
	safePath, err := getSafePath(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	fileInfo, err := os.Stat(safePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if fileInfo.IsDir() {
		http.Error(w, "Path is a directory", http.StatusBadRequest)
		return
	}

	// Read content
	content, err := os.ReadFile(safePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	// Detect if binary
	isBinary := false
	if len(content) > 0 {
		// Sample first 512 bytes
		sample := content
		if len(sample) > 512 {
			sample = sample[:512]
		}
		
		// If it contains null bytes, treat as binary
		for _, b := range sample {
			if b == 0 {
				isBinary = true
				break
			}
		}

		// Also check UTF-8 validity of sample if not already flagged as binary
		if !isBinary && !utf8.Valid(sample) {
			isBinary = true
		}
	}

	if isBinary {
		// Return binary indicator or serve file with detected content type
		// If it's an image, we can serve it directly, otherwise we return a 200 JSON indicating binary
		mimeType := http.DetectContentType(content)
		if strings.HasPrefix(mimeType, "image/") {
			w.Header().Set("Content-Type", mimeType)
			w.Write(content)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"isBinary": true,
			"mimeType": mimeType,
			"size":     fileInfo.Size(),
		})
		return
	}

	// Text file
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]SearchResult{})
		return
	}

	var results []SearchResult
	maxResults := 200 // Avoid overwhelming search
	count := 0

	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if count >= maxResults {
			return filepath.SkipDir // Stop walking once we hit the limit
		}

		// Skip directories we don't care about
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == ".svelte-kit" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files larger than 1MB for content search to keep it fast
		if info.Size() > 1024*1024 {
			return nil
		}

		// Read file contents to search
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Check if it's binary
		for _, b := range content {
			if b == 0 {
				return nil // skip binary files
			}
		}

		lines := strings.Split(string(content), "\n")
		relPath, _ := filepath.Rel(workspaceDir, path)
		relPath = filepath.ToSlash(relPath)

		// Search in filepath name
		pathMatched := strings.Contains(strings.ToLower(relPath), strings.ToLower(query))

		for idx, line := range lines {
			if count >= maxResults {
				break
			}
			// Search in line content
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				results = append(results, SearchResult{
					Path:        relPath,
					LineNumber:  idx + 1,
					LineContent: strings.TrimSpace(line),
				})
				count++
			}
		}

		// If path matched but no line content matches, we can add a dummy match or just skip.
		// Let's add it if we haven't found any content matches for this file but path matches.
		if pathMatched && count < maxResults {
			// Find if we already added matches for this file
			hasMatch := false
			for _, res := range results {
				if res.Path == relPath {
					hasMatch = true
					break
				}
			}
			if !hasMatch {
				results = append(results, SearchResult{
					Path:        relPath,
					LineNumber:  0,
					LineContent: "[File name match]",
				})
				count++
			}
		}

		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	var files []string
	maxFiles := 1000
	count := 0

	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors
		}
		if count >= maxFiles {
			return filepath.SkipDir
		}

		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == ".svelte-kit" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return nil
		}

		files = append(files, filepath.ToSlash(relPath))
		count++
		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list files: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

type GitInfo struct {
	IsGit         bool     `json:"isGit"`
	CurrentBranch string   `json:"currentBranch"`
	Branches      []string `json:"branches"`
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workspaceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func handleGitBranches(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		json.NewEncoder(w).Encode(GitInfo{IsGit: false})
		return
	}

	// Check if it's a git repo
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workspaceDir
	if err := cmd.Run(); err != nil {
		json.NewEncoder(w).Encode(GitInfo{IsGit: false})
		return
	}

	// Get current branch
	currentBranch, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		currentBranch = "HEAD"
	}

	// Get all branches
	branchesOut, err := runGitCommand("branch", "-a", "--format=%(refname:short)")
	var branches []string
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

	json.NewEncoder(w).Encode(GitInfo{
		IsGit:         true,
		CurrentBranch: currentBranch,
		Branches:      branches,
	})
}

func handleGitCheckout(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	branch := r.URL.Query().Get("branch")
	if branch == "" {
		http.Error(w, "Missing branch parameter", http.StatusBadRequest)
		return
	}

	// Validate branch name to prevent malicious commands or path injection
	branchesOut, err := runGitCommand("branch", "-a", "--format=%(refname:short)")
	if err != nil {
		http.Error(w, "Failed to fetch branches: "+err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, "Invalid branch name", http.StatusBadRequest)
		return
	}

	_, err = runGitCommand("checkout", branch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleLSP(ws *websocket.Conn) {
	defer ws.Close()

	lang := ws.Request().URL.Query().Get("lang")
	log.Printf("LSP client connected over WebSocket. Language: %s", lang)

	var cmdPath string
	var cmdArgs []string

	switch lang {
	case "go":
		goplsPath, err := exec.LookPath("gopls")
		if err != nil {
			goplsPath = "/home/ucok/.local/share/mise/installs/go/1.26.1/bin/gopls"
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
		tsLspPath := filepath.Join(workspaceDir, "frontend", "node_modules", ".bin", "typescript-language-server")
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
		return
	}

	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Dir = workspaceDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("LSP StdinPipe failed: %v", err)
		return
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("LSP StdoutPipe failed: %v", err)
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		log.Printf("LSP process start failed (%s): %v", cmdPath, err)
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

	// Read gopls stdout -> write to WebSocket
	go func() {
		defer closeDone()
		reader := bufio.NewReader(stdout)
		for {
			var contentLength int
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					log.Printf("LSP reader error: %v", err)
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

	// Read WebSocket -> write to gopls stdin
	go func() {
		defer closeDone()
		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				log.Printf("LSP ws receive failed: %v", err)
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

