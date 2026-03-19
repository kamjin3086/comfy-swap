package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultPort    = "8189"
	DefaultDataDir = "./data"
)

type ServerConfig struct {
	Port    string
	DataDir string
}

type Settings struct {
	ComfyUIURL string    `json:"comfyui_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ResolveDataPaths(dataDir string) (settingsPath, workflowsDir string) {
	return filepath.Join(dataDir, "settings.json"), filepath.Join(dataDir, "workflows")
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
