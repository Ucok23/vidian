package goreview

import (
	"encoding/json"
	"net/http"

	"github.com/Ucok23/vidian/internal/config"
)

// HandleStatus reports whether the workspace is a Go module, so the frontend
// knows whether to show the Go Review tab at all.
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	ws := resolveWorkspace(w, r)
	if ws == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CheckStatus(ws.Path))
}

// HandleRun executes the full deterministic review and returns the report.
func HandleRun(w http.ResponseWriter, r *http.Request) {
	setupCORS(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	ws := resolveWorkspace(w, r)
	if ws == nil {
		return
	}
	report, err := Run(ws.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// --- local copies of server's small unexported helpers ---
// (internal/server already has these but they're unexported to that
// package; goreview is called from server via HTTP registration only, so it
// needs its own copies rather than an import cycle.)

func setupCORS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func resolveWorkspace(w http.ResponseWriter, r *http.Request) *config.Workspace {
	id := r.URL.Query().Get("ws")
	if id == "" {
		http.Error(w, "missing workspace parameter", http.StatusBadRequest)
		return nil
	}
	ws := config.ActiveConfig.Get(id)
	if ws == nil {
		http.Error(w, "unknown workspace: "+id, http.StatusBadRequest)
		return nil
	}
	return ws
}
