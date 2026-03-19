package cmd

import (
	"errors"

	"comfy-swap/internal/client"

	"github.com/spf13/cobra"
)

func init() {
	var save string
	cmd := &cobra.Command{
		Use:   "result <prompt_id>",
		Short: "Fetch completed prompt result",
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
			outputs := extractOutputs(h, args[0])
			if save == "" {
				return printResult(map[string]interface{}{
					"prompt_id": args[0],
					"outputs":   outputs,
				})
			}
			files, err := downloadOutputs(c, outputs, save)
			if err != nil {
				return err
			}
			if flagQuiet {
				if len(files) == 1 {
					return printResult(files[0])
				}
				return printResult(files)
			}
			return printResult(map[string]interface{}{
				"prompt_id": args[0],
				"files":     files,
			})
		},
	}
	cmd.Flags().StringVar(&save, "save", "", "save to directory or file path")
	rootCmd.AddCommand(cmd)
}
