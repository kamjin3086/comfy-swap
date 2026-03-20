package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var version = "0.1.6"

func init() {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(map[string]string{
					"version":    version,
					"go_version": runtime.Version(),
					"os":         runtime.GOOS,
					"arch":       runtime.GOARCH,
				})
			} else {
				fmt.Printf("comfy-swap v%s (%s, %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(cmd)
}
