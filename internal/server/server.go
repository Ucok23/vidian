package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/websocket"
	_ "modernc.org/sqlite"

	"github.com/Ucok23/vidian/internal/config"
	"github.com/Ucok23/vidian/internal/git"
	"github.com/Ucok23/vidian/internal/lsp"
)

type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
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

func setupCORS(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Start registers routes and starts the HTTP listener
func Start(cfg *config.Config, embeddedFiles fs.FS) {
	config.ActiveConfig = cfg

	// Register API endpoints
	http.HandleFunc("/api/workspace", handleWorkspace)
	http.HandleFunc("/api/dir", handleDir)
	http.HandleFunc("/api/file", handleFile)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/files", handleFiles)
	http.HandleFunc("/api/git/branches", handleGitBranches)
	http.HandleFunc("/api/git/checkout", handleGitCheckout)
	http.HandleFunc("/api/git/changes", handleGitChanges)
	http.HandleFunc("/api/git/show", handleGitShow)
	http.HandleFunc("/api/git/blame", handleGitBlame)
	http.HandleFunc("/api/git/log", handleGitLog)
	http.HandleFunc("/api/git/commit/files", handleGitCommitFiles)
	http.HandleFunc("/api/git/commit", handleGitCommit)
	http.HandleFunc("/api/sqlite/tables", handleSQLiteTables)
	http.HandleFunc("/api/sqlite/query", handleSQLiteQuery)
	http.Handle("/api/lsp", websocket.Handler(lsp.HandleLSP))

	// Register Frontend serving
	if cfg.DevMode {
		frontendDist := "./frontend/dist"
		log.Printf("Serving static assets from local path: %s", frontendDist)
		fileServer := http.FileServer(http.Dir(frontendDist))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(frontendDist, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(frontendDist, "index.html"))
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	} else {
		log.Printf("Serving embedded static assets")
		distFS, err := fs.Sub(embeddedFiles, "frontend/dist")
		if err != nil {
			log.Fatalf("Error accessing embedded assets: %v", err)
		}
		fileServer := http.FileServer(http.FS(distFS))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			_, err := distFS.Open(path)
			if err != nil {
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

	log.Printf("Server listening on http://localhost:%d", cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil))
}

func handleWorkspace(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	info := WorkspaceInfo{
		Name: filepath.Base(config.ActiveConfig.WorkspaceDir),
		Path: config.ActiveConfig.WorkspaceDir,
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
	safePath, err := config.GetSafePath(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read directory: %v", err), http.StatusInternalServerError)
		return
	}

	var files []FileInfo = []FileInfo{}
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" || name == "node_modules" || name == ".DS_Store" {
			continue
		}

		info, err := entry.Info()
		var size int64
		if err == nil {
			size = info.Size()
		}

		entryRelPath := filepath.Join(relPath, name)
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
	safePath, err := config.GetSafePath(relPath)
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

	content, err := os.ReadFile(safePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	isBinary := false
	if len(content) > 0 {
		sample := content
		if len(sample) > 512 {
			sample = sample[:512]
		}
		for _, b := range sample {
			if b == 0 {
				isBinary = true
				break
			}
		}
		if !isBinary && !utf8.Valid(sample) {
			isBinary = true
		}
	}

	if isBinary {
		mimeType := http.DetectContentType(content)
		ext := strings.ToLower(filepath.Ext(safePath))
		videoMimes := map[string]string{
			".mp4": "video/mp4", ".webm": "video/webm", ".ogg": "video/ogg", ".mov": "video/quicktime",
		}
		audioMimes := map[string]string{
			".mp3": "audio/mpeg", ".wav": "audio/wav", ".flac": "audio/flac", ".ogg": "audio/ogg", ".m4a": "audio/mp4",
		}
		if vm, ok := videoMimes[ext]; ok {
			w.Header().Set("Content-Type", vm)
			w.Write(content)
			return
		}
		if am, ok := audioMimes[ext]; ok {
			w.Header().Set("Content-Type", am)
			w.Write(content)
			return
		}
		if strings.HasPrefix(mimeType, "image/") {
			w.Header().Set("Content-Type", mimeType)
			w.Write(content)
			return
		}

		sqliteExts := map[string]bool{".db": true, ".sqlite": true, ".sqlite3": true}
		if sqliteExts[ext] {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"isSQLite": true,
				"path":     relPath,
				"size":     fileInfo.Size(),
			})
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

	var results []SearchResult = []SearchResult{}
	maxResults := 200
	count := 0

	err := filepath.Walk(config.ActiveConfig.WorkspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if count >= maxResults {
			return filepath.SkipDir
		}

		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == ".svelte-kit" {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Size() > 1024*1024 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		for _, b := range content {
			if b == 0 {
				return nil
			}
		}

		lines := strings.Split(string(content), "\n")
		relPath, _ := filepath.Rel(config.ActiveConfig.WorkspaceDir, path)
		relPath = filepath.ToSlash(relPath)
		pathMatched := strings.Contains(strings.ToLower(relPath), strings.ToLower(query))

		for idx, line := range lines {
			if count >= maxResults {
				break
			}
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				results = append(results, SearchResult{
					Path:        relPath,
					LineNumber:  idx + 1,
					LineContent: strings.TrimSpace(line),
				})
				count++
			}
		}

		if pathMatched && count < maxResults {
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

	var files []string = []string{}
	maxFiles := 1000
	count := 0

	err := filepath.Walk(config.ActiveConfig.WorkspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
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

		relPath, err := filepath.Rel(config.ActiveConfig.WorkspaceDir, path)
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

func handleGitBranches(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	w.Header().Set("Content-Type", "application/json")

	info, err := git.GetBranches()
	if err != nil {
		json.NewEncoder(w).Encode(git.GitInfo{IsGit: false})
		return
	}
	json.NewEncoder(w).Encode(info)
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

	err := git.Checkout(branch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleGitChanges(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	w.Header().Set("Content-Type", "application/json")

	changes, err := git.GetChanges()
	if err != nil {
		json.NewEncoder(w).Encode([]git.GitChange{})
		return
	}
	json.NewEncoder(w).Encode(changes)
}

func handleGitShow(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	path := r.URL.Query().Get("path")
	commit := r.URL.Query().Get("commit")
	if path == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	out, err := git.Show(path, commit)
	if err != nil {
		http.Error(w, "File not found in git: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(out))
}

func handleGitBlame(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	blameLines, err := git.Blame(path)
	if err != nil {
		json.NewEncoder(w).Encode([]git.BlameLine{})
		return
	}
	json.NewEncoder(w).Encode(blameLines)
}

func handleGitLog(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	path := r.URL.Query().Get("path")
	w.Header().Set("Content-Type", "application/json")

	commits, err := git.Log(path)
	if err != nil {
		json.NewEncoder(w).Encode([]git.CommitInfo{})
		return
	}
	json.NewEncoder(w).Encode(commits)
}

func handleGitCommitFiles(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	commit := r.URL.Query().Get("commit")
	if commit == "" {
		http.Error(w, "Missing commit parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	changes, err := git.GetCommitFiles(commit)
	if err != nil {
		json.NewEncoder(w).Encode([]git.GitChange{})
		return
	}
	json.NewEncoder(w).Encode(changes)
}

func handleGitCommit(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	details, err := git.GetCommitDetails(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(details)
}

var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func handleSQLiteTables(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	relPath := r.URL.Query().Get("path")
	safePath, err := config.GetSafePath(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	db, err := sql.Open("sqlite", safePath+"?mode=ro")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query tables: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tables)
}

func handleSQLiteQuery(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	relPath := r.URL.Query().Get("path")
	safePath, err := config.GetSafePath(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	table := r.URL.Query().Get("table")
	if !validTableName.MatchString(table) {
		http.Error(w, "Invalid table name", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	db, err := sql.Open("sqlite", safePath+"?mode=ro")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get total row count
	var totalRows int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)).Scan(&totalRows)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to count rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Query data
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d OFFSET %d", table, limit, offset))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query table: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get columns: %v", err), http.StatusInternalServerError)
		return
	}

	var data [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := make([]interface{}, len(columns))
		for i, v := range values {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			default:
				row[i] = val
			}
		}
		data = append(data, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"columns":   columns,
		"rows":      data,
		"totalRows": totalRows,
	})
}
