package cmd

import (
	"errors"

	"comfy-swap/internal/client"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "status <prompt_id>",
		Short: "Get prompt status",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("prompt_id is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(flagServer)
			h, err := c.GetHistory(args[0])
			if err != nil {
				return err
			}
			status := "running"
			if _, ok := h[args[0]]; ok {
				status = "completed"
			}
			if flagQuiet {
				return printResult(status)
			}
			return printResult(map[string]interface{}{
				"prompt_id": args[0],
				"status":    status,
			})
		},
	}
	rootCmd.AddCommand(cmd)
}
