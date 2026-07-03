package config

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Workspace is a single directory opened in Vidian. Multiple workspaces are
// served by one running instance and addressed per-request by ID.
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"` // absolute
}

// Config is the root server configuration. Prefer New(); ActiveConfig is
// retained for backward compatibility.
type Config struct {
	mu         sync.RWMutex
	workspaces map[string]*Workspace
	DevMode    bool
	Port       int
}

// ActiveConfig is the global configuration reference initialized on server
// start. It is retained for backward compatibility; new servers should pass
// configs explicitly.
var ActiveConfig *Config

// New creates an empty Config ready to have workspaces added.
func New(devMode bool, port int) *Config {
	return &Config{
		workspaces: make(map[string]*Workspace),
		DevMode:    devMode,
		Port:       port,
	}
}

var slugCleaner = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// workspaceID derives a stable, collision-resistant ID from an absolute path:
// the base name (slugified) plus a short hash of the full path. Two different
// directories that share a base name get distinct IDs, and the same directory
// always maps to the same ID.
func workspaceID(absPath string) string {
	base := slugCleaner.ReplaceAllString(filepath.Base(absPath), "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "ws"
	}
	sum := sha1.Sum([]byte(absPath))
	return base + "-" + hex.EncodeToString(sum[:])[:6]
}

// Add registers a workspace for the given absolute path. It is idempotent:
// re-adding a path that is already registered returns the existing workspace.
func (c *Config) Add(absPath string) *Workspace {
	id := workspaceID(absPath)
	c.mu.Lock()
	defer c.mu.Unlock()
	if ws, ok := c.workspaces[id]; ok {
		return ws
	}
	ws := &Workspace{
		ID:   id,
		Name: filepath.Base(absPath),
		Path: absPath,
	}
	c.workspaces[id] = ws
	return ws
}

// Get returns the workspace with the given ID, or nil if not registered.
func (c *Config) Get(id string) *Workspace {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.workspaces[id]
}

// List returns all registered workspaces sorted by name.
func (c *Config) List() []*Workspace {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*Workspace, 0, len(c.workspaces))
	for _, ws := range c.workspaces {
		out = append(out, ws)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Path < out[j].Path
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// GetSafePath cleans and validates a requested path against a workspace root to
// prevent directory traversal attacks.
func GetSafePath(ws *Workspace, reqPath string) (string, error) {
	if ws == nil {
		return "", errors.New("workspace not specified")
	}
	targetPath := filepath.Clean(filepath.Join(ws.Path, reqPath))
	rel, err := filepath.Rel(ws.Path, targetPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("access denied: path lies outside workspace root")
	}
	return targetPath, nil
}

// GetSafePathFor is the explicit config-aware variant of GetSafePath and
// accepts the owner Config directly. For call sites that already have the
// workspace pointer, use GetSafePath instead.
func GetSafePathFor(cfg *Config, reqPath string) (string, error) {
	if cfg == nil {
		return "", errors.New("configuration not initialized")
	}
	ws := cfg.List()[0]
	if ws == nil {
		return "", errors.New("no workspace registered")
	}
	return GetSafePath(ws, reqPath)
}

// Settings holds user-level preferences that persist across restarts and are
// independent of any workspace — currently just the BYO Anthropic API key
// used by the AI onboarding narrator. Never committed to a repo.
type Settings struct {
	AnthropicAPIKey string `json:"anthropicApiKey,omitempty"`
}

// settingsPathOverride lets tests redirect settings I/O to a temp dir.
var settingsPathOverride string

func settingsPath() (string, error) {
	if settingsPathOverride != "" {
		return settingsPathOverride, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vidian", "settings.json"), nil
}

// LoadSettings reads Settings from disk, returning a zero-value Settings if
// none has been saved yet.
func LoadSettings() (*Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return &Settings{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return &Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return &Settings{}, err
	}
	return &s, nil
}

// SaveSettings persists Settings to disk with owner-only permissions, since
// it may contain an API key.
func SaveSettings(s *Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
