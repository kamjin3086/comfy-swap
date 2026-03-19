package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func init() {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("comfy-swap v%s (%s, %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
	rootCmd.AddCommand(cmd)
}
