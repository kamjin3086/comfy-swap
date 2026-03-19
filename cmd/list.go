package cmd

import (
	"comfy-swap/internal/client"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(flagServer)
			items, err := c.ListWorkflows()
			if err != nil {
				return err
			}
			if flagQuiet {
				lines := make([]string, 0, len(items))
				for _, it := range items {
					lines = append(lines, it.ID)
				}
				return printResult(lines)
			}
			return printResult(items)
		},
	}
	rootCmd.AddCommand(cmd)
}
