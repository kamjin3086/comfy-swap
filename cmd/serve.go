package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"comfy-swap/internal/assets"
	"comfy-swap/internal/config"
	"comfy-swap/internal/server"

	"github.com/spf13/cobra"
)

func init() {
	var port string
	var dataDir string
	var daemon bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start comfy-swap server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dataDir == "" {
				dataDir = config.DefaultDataDir()
			}

			if daemon {
				return startDaemon(port, dataDir)
			}

			if err := config.EnsureDataDir(dataDir); err != nil {
				return err
			}
			app, err := server.New(dataDir, assets.WebFS)
			if err != nil {
				return err
			}
			addr := ":" + port
			fmt.Printf("comfy-swap server listening on %s\n", addr)
			fmt.Printf("Data directory: %s\n", dataDir)
			return http.ListenAndServe(addr, app.Router())
		},
	}
	cmd.Flags().StringVar(&port, "port", config.DefaultPort, "listen port")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data directory (default: OS-specific location, see --help)")
	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run as background daemon")
	rootCmd.AddCommand(cmd)
}

func startDaemon(port, dataDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{"serve", "--port", port}
	if dataDir != "" {
		args = append(args, "--data-dir", dataDir)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if runtime.GOOS != "windows" {
		cmd.Env = append(os.Environ(), "COMFY_SWAP_DAEMON=1")
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Printf("comfy-swap daemon started (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("Server: http://localhost:%s\n", port)
	fmt.Printf("Stop: comfy-swap stop (or kill %d)\n", cmd.Process.Pid)
	return nil
}
