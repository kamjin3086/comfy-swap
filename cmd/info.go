package cmd

import (
	"errors"

	"comfy-swap/internal/client"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "info <workflow_id>",
		Short: "Get workflow details",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("workflow_id is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(flagServer)
			wf, err := c.GetWorkflow(args[0])
			if err != nil {
				return err
			}
			return printResult(wf)
		},
	}
	rootCmd.AddCommand(cmd)
}
