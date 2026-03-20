package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	DefaultPort = "8189"
)

// DefaultDataDir returns the default data directory based on OS conventions.
// Priority: COMFY_SWAP_DATA env > OS-specific default location
//   - Windows: %APPDATA%\comfy-swap
//   - macOS: ~/Library/Application Support/comfy-swap
//   - Linux: ~/.local/share/comfy-swap (XDG_DATA_HOME)
func DefaultDataDir() string {
	if envDir := os.Getenv("COMFY_SWAP_DATA"); envDir != "" {
		return envDir
	}

	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "comfy-swap")
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "comfy-swap")
		}
	default: // linux and others
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return filepath.Join(xdgData, "comfy-swap")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", "comfy-swap")
		}
	}

	// Fallback to current directory
	return "./data"
}

type ServerConfig struct {
	Port    string
	DataDir string
}

type Settings struct {
	ComfyUIURL       string    `json:"comfyui_url"`
	LogRetentionDays int       `json:"log_retention_days"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (s *Settings) GetLogRetentionDays() int {
	if s == nil || s.LogRetentionDays <= 0 {
		return 7
	}
	return s.LogRetentionDays
}

func ResolveDataPaths(dataDir string) (settingsPath, workflowsDir string) {
	return filepath.Join(dataDir, "settings.json"), filepath.Join(dataDir, "workflows")
}

func ResolveLogsDir(dataDir string) string {
	return filepath.Join(dataDir, "logs")
}

func EnsureDataDir(dataDir string) error {
	return os.MkdirAll(filepath.Join(dataDir, "workflows"), 0o755)
}

func LoadSettings(settingsPath string) (*Settings, error) {
	b, err := os.ReadFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveSettings(settingsPath string, s *Settings) error {
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, b, 0o644)
}
