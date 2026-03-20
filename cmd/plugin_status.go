package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "plugin-status",
		Short: "Check if ComfyUI plugin is installed and connected",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(flagServer + "/api/plugin-status")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
			}

			var status map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				return err
			}

			if flagQuiet {
				if s, ok := status["status"].(string); ok {
					fmt.Println(s)
				}
				return nil
			}
			return printResult(status)
		},
	}
	rootCmd.AddCommand(cmd)
}
