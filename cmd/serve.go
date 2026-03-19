package cmd

import (
	"fmt"
	"net/http"

	"comfy-swap/internal/assets"
	"comfy-swap/internal/config"
	"comfy-swap/internal/server"

	"github.com/spf13/cobra"
)

func init() {
	var port string
	var dataDir string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start comfy-swap server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureDataDir(dataDir); err != nil {
				return err
			}
			app, err := server.New(dataDir, assets.WebFS, assets.PluginFS)
			if err != nil {
				return err
			}
			addr := ":" + port
			fmt.Printf("comfy-swap server listening on %s\n", addr)
			return http.ListenAndServe(addr, app.Router())
		},
	}
	cmd.Flags().StringVar(&port, "port", config.DefaultPort, "listen port")
	cmd.Flags().StringVar(&dataDir, "data-dir", config.DefaultDataDir, "data directory")
	rootCmd.AddCommand(cmd)
}
