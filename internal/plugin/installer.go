package plugin

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	PluginRepoURL  = "https://github.com/kamjin3086/ComfyUI-ComfySwap"
	PluginZipURL   = "https://github.com/kamjin3086/ComfyUI-ComfySwap/archive/refs/heads/main.zip"
	PluginDirName  = "ComfyUI-ComfySwap"
	DefaultTimeout = 60 * time.Second
)

type InstallResult struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

func Install(targetCustomNodesDir string) (*InstallResult, error) {
	targetDir := filepath.Join(targetCustomNodesDir, PluginDirName)

	if _, err := exec.LookPath("git"); err == nil {
		if err := installViaGit(targetDir); err == nil {
			return &InstallResult{Path: targetDir, Method: "git"}, nil
		}
	}

	if err := installViaZip(targetDir); err != nil {
		return nil, err
	}
	return &InstallResult{Path: targetDir, Method: "zip"}, nil
}

func installViaGit(targetDir string) error {
	if _, err := os.Stat(targetDir); err == nil {
		cmd := exec.Command("git", "-C", targetDir, "pull", "--ff-only")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("git", "clone", "--depth", "1", PluginRepoURL+".git", targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installViaZip(targetDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, PluginZipURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download plugin: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read plugin zip: %w", err)
	}

	return extractZip(data, targetDir)
}

func extractZip(data []byte, targetDir string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	if err := os.RemoveAll(targetDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	var prefix string
	for _, f := range zr.File {
		if f.FileInfo().IsDir() && strings.Count(f.Name, "/") == 1 {
			prefix = f.Name
			break
		}
	}

	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		relPath := strings.TrimPrefix(f.Name, prefix)
		if relPath == "" {
			continue
		}

		target := filepath.Join(targetDir, filepath.Clean(relPath))
		if !strings.HasPrefix(target, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func GetDownloadURL() string {
	return PluginZipURL
}

func GetRepoURL() string {
	return PluginRepoURL
}
