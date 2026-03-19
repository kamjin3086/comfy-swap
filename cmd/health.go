package cmd

import (
	"comfy-swap/internal/client"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check comfy-swap and ComfyUI health",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(flagServer)
			h, err := c.Health()
			if err != nil {
				return err
			}
			if flagQuiet {
				return printResult(h.Status)
			}
			return printResult(h)
		},
	}
	rootCmd.AddCommand(cmd)
}
