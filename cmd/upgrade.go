package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	"comfy-swap/internal/upgrade"

	"github.com/spf13/cobra"
)

func init() {
	var checkOnly bool
	var downloadOnly bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for updates and download new version",
		Long: `Check for new versions and download updates.

This command is designed to be AI-agent friendly with predictable JSON outputs.

Typical upgrade flow:
  1. comfy-swap upgrade --check --json    # Check if update available
  2. comfy-swap upgrade --download --json # Download new binary to temp
  3. Replace binary manually (agent executes shell command)
  4. comfy-swap version --json            # Verify upgrade succeeded

Examples:
  # Check for updates (human readable)
  comfy-swap upgrade --check

  # Check for updates (JSON for automation)
  comfy-swap upgrade --check --json

  # Download new version (does not replace current binary)
  comfy-swap upgrade --download --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkOnly {
				return runCheck(jsonOutput)
			}
			if downloadOnly {
				return runDownload(jsonOutput)
			}
			// Default: just check
			return runCheck(jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check if a new version is available")
	cmd.Flags().BoolVar(&downloadOnly, "download", false, "Download new version to temp directory")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for automation)")

	rootCmd.AddCommand(cmd)
}

func runCheck(jsonOutput bool) error {
	result, err := upgrade.CheckLatestVersion(version)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		if err != nil {
			os.Exit(1)
		}
		return nil
	}

	// Human readable output
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		return err
	}

	fmt.Printf("Current version: %s\n", result.Current)
	fmt.Printf("Latest version:  %s\n", result.Latest)

	if result.UpdateAvailable {
		fmt.Println()
		fmt.Println("A new version is available!")
		fmt.Printf("Download: %s\n", result.DownloadURL)
		fmt.Printf("Release:  %s\n", result.ReleaseURL)
		fmt.Println()
		fmt.Println("To upgrade, run:")
		fmt.Println("  comfy-swap upgrade --download")
	} else {
		fmt.Println()
		fmt.Println("You are running the latest version.")
	}

	return nil
}

func runDownload(jsonOutput bool) error {
	// First check for updates
	checkResult, err := upgrade.CheckLatestVersion(version)
	if err != nil {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("failed to check version: %v", err),
			})
		} else {
			fmt.Printf("Error checking for updates: %v\n", err)
		}
		return err
	}

	if !checkResult.UpdateAvailable {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"success":          true,
				"update_available": false,
				"message":          "already at latest version",
				"current":          checkResult.Current,
			})
		} else {
			fmt.Printf("You are running the latest version (%s).\n", checkResult.Current)
		}
		return nil
	}

	if checkResult.DownloadURL == "" {
		msg := fmt.Sprintf("no download available for %s/%s", runtime.GOOS, runtime.GOARCH)
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"success": false,
				"error":   msg,
			})
		} else {
			fmt.Printf("Error: %s\n", msg)
		}
		return errors.New(msg)
	}

	if !jsonOutput {
		fmt.Printf("Downloading v%s...\n", checkResult.Latest)
	}

	downloadResult, err := upgrade.Download(checkResult.DownloadURL, checkResult.SHA256URL)
	if err != nil {
		if jsonOutput {
			outputJSON(downloadResult)
		} else {
			fmt.Printf("Download failed: %v\n", err)
		}
		return err
	}

	// Add version info to result
	if jsonOutput {
		outputJSON(map[string]interface{}{
			"success":       downloadResult.Success,
			"download_path": downloadResult.DownloadPath,
			"sha256":        downloadResult.SHA256,
			"expected_hash": downloadResult.ExpectedHash,
			"hash_verified": downloadResult.HashVerified,
			"version":       checkResult.Latest,
			"current_exe":   getCurrentExePath(),
		})
	} else {
		fmt.Println()
		fmt.Println("Download complete!")
		fmt.Printf("  New binary: %s\n", downloadResult.DownloadPath)
		fmt.Printf("  SHA256:     %s\n", downloadResult.SHA256)
		if downloadResult.ExpectedHash != "" {
			if downloadResult.HashVerified {
				fmt.Println("  Hash:       ✓ Verified")
			} else {
				fmt.Println("  Hash:       ✗ MISMATCH (expected: " + downloadResult.ExpectedHash + ")")
			}
		}
		fmt.Println()
		fmt.Println("To complete the upgrade, replace the current binary:")
		currentExe := getCurrentExePath()
		if runtime.GOOS == "windows" {
			fmt.Printf("  Move-Item -Force \"%s\" \"%s\"\n", downloadResult.DownloadPath, currentExe)
		} else {
			fmt.Printf("  sudo mv \"%s\" \"%s\"\n", downloadResult.DownloadPath, currentExe)
		}
		fmt.Println()
		fmt.Println("Then verify:")
		fmt.Println("  comfy-swap version")
	}

	return nil
}

func getCurrentExePath() string {
	exe, err := upgrade.GetCurrentExecutablePath()
	if err != nil {
		return "comfy-swap"
	}
	return exe
}

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}
