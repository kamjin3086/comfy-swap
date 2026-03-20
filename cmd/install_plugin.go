package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"comfy-swap/internal/plugin"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "install-plugin <custom_nodes_path>",
		Short: "Install ComfyUI-ComfySwap plugin from GitHub",
		Long: `Install the ComfyUI-ComfySwap plugin to enable workflow export.

The plugin is downloaded from GitHub:
  ` + plugin.GetRepoURL() + `

Installation methods (in order of preference):
  1. Git clone (if git is available)
  2. Download ZIP and extract`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("custom_nodes path is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			st, err := os.Stat(target)
			if err != nil {
				return fmt.Errorf("cannot access %s: %w", target, err)
			}
			if !st.IsDir() {
				return fmt.Errorf("%s is not a directory", target)
			}

			fmt.Printf("Installing ComfyUI-ComfySwap plugin to %s...\n", target)
			result, err := plugin.Install(target)
			if err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			absPath, _ := filepath.Abs(result.Path)
			fmt.Printf("Installed via %s to %s\n", result.Method, absPath)
			fmt.Println("Please restart or refresh ComfyUI to load the plugin.")
			return nil
		},
	}
	rootCmd.AddCommand(cmd)
}
