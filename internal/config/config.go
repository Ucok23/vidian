package config

import (
	"errors"
	"path/filepath"
	"strings"
)

type Config struct {
	WorkspaceDir string
	DevMode      bool
	Port         int
}

// ActiveConfig is the global configuration reference initialized on server start
var ActiveConfig *Config

// GetSafePath cleans and validates requested paths against the active workspace root to prevent directory traversal attacks
func GetSafePath(reqPath string) (string, error) {
	if ActiveConfig == nil {
		return "", errors.New("configuration not initialized")
	}
	targetPath := filepath.Clean(filepath.Join(ActiveConfig.WorkspaceDir, reqPath))
	rel, err := filepath.Rel(ActiveConfig.WorkspaceDir, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("access denied: path lies outside workspace root")
	}
	return targetPath, nil
}
