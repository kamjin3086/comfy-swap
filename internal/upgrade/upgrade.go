package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	GitHubOwner = "kamjin3086"
	GitHubRepo  = "comfy-swap"
	ReleasesAPI = "https://api.github.com/repos/" + GitHubOwner + "/" + GitHubRepo + "/releases/latest"
)

type ReleaseInfo struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Assets     []Asset `json:"assets"`
	HTMLURL    string  `json:"html_url"`
	Prerelease bool    `json:"prerelease"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type CheckResult struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	DownloadURL     string `json:"download_url,omitempty"`
	SHA256URL       string `json:"sha256_url,omitempty"`
	ReleaseURL      string `json:"release_url,omitempty"`
	AssetName       string `json:"asset_name,omitempty"`
	Error           string `json:"error,omitempty"`
}

type DownloadResult struct {
	Success      bool   `json:"success"`
	DownloadPath string `json:"download_path,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	ExpectedHash string `json:"expected_hash,omitempty"`
	HashVerified bool   `json:"hash_verified"`
	Error        string `json:"error,omitempty"`
}

func CheckLatestVersion(currentVersion string) (*CheckResult, error) {
	result := &CheckResult{
		Current: currentVersion,
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", ReleasesAPI, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "comfy-swap-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch release info: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("GitHub API returned status %d", resp.StatusCode)
		return result, errors.New(result.Error)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		result.Error = fmt.Sprintf("failed to parse release info: %v", err)
		return result, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	result.Latest = latestVersion
	result.ReleaseURL = release.HTMLURL
	result.UpdateAvailable = compareVersions(currentVersion, latestVersion) < 0

	assetName := getAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.DownloadURL = asset.BrowserDownloadURL
			result.AssetName = asset.Name
		}
		if asset.Name == assetName+".sha256" {
			result.SHA256URL = asset.BrowserDownloadURL
		}
	}

	if result.DownloadURL == "" && result.UpdateAvailable {
		result.Error = fmt.Sprintf("no asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return result, nil
}

func Download(downloadURL, sha256URL string) (*DownloadResult, error) {
	result := &DownloadResult{}

	tmpDir := os.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	downloadPath := filepath.Join(tmpDir, fmt.Sprintf("comfy-swap-new%s", ext))

	client := &http.Client{Timeout: 5 * time.Minute}

	// Download binary
	resp, err := client.Get(downloadURL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to download: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("download returned status %d", resp.StatusCode)
		return result, errors.New(result.Error)
	}

	outFile, err := os.Create(downloadPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create file: %v", err)
		return result, err
	}

	hasher := sha256.New()
	writer := io.MultiWriter(outFile, hasher)

	_, err = io.Copy(writer, resp.Body)
	outFile.Close()
	if err != nil {
		os.Remove(downloadPath)
		result.Error = fmt.Sprintf("failed to write file: %v", err)
		return result, err
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		os.Chmod(downloadPath, 0755)
	}

	result.DownloadPath = downloadPath
	result.SHA256 = hex.EncodeToString(hasher.Sum(nil))

	// Verify hash if available
	if sha256URL != "" {
		expectedHash, err := fetchExpectedHash(client, sha256URL)
		if err == nil {
			result.ExpectedHash = expectedHash
			result.HashVerified = strings.EqualFold(result.SHA256, expectedHash)
		}
	}

	result.Success = true
	return result, nil
}

func fetchExpectedHash(client *http.Client, sha256URL string) (string, error) {
	resp, err := client.Get(sha256URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Format: "hash  filename" or just "hash"
	parts := strings.Fields(string(body))
	if len(parts) > 0 {
		return strings.ToLower(parts[0]), nil
	}
	return "", fmt.Errorf("empty hash file")
}

func getAssetName() string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("comfy-swap-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

func compareVersions(v1, v2 string) int {
	// Simple version comparison: split by dots and compare numerically
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &n2)
		}
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}
	return 0
}

// GetCurrentExecutablePath returns the path of the currently running executable
func GetCurrentExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}
