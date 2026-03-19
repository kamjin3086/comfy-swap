package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"comfy-swap/internal/assets"
	"comfy-swap/internal/server"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "install-plugin <custom_nodes_path>",
		Short: "Install comfy-swap ComfyUI plugin",
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
				return err
			}
			if !st.IsDir() {
				return fmt.Errorf("%s is not a directory", target)
			}
			installer := &server.Installer{PluginFS: assets.PluginFS}
			out, err := installer.InstallPlugin(target)
			if err != nil {
				return err
			}
			out, _ = filepath.Abs(out)
			fmt.Printf("Installed comfy-swap plugin to %s\n", out)
			fmt.Println("Please refresh ComfyUI page to load the plugin.")
			return nil
		},
	}
	rootCmd.AddCommand(cmd)
}
