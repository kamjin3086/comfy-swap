package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	var workflowID string
	var limit int
	var offset int

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View request logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoint := flagServer + "/api/logs"
			if workflowID != "" {
				endpoint = flagServer + "/api/logs/" + url.PathEscape(workflowID)
			}

			q := url.Values{}
			if limit > 0 {
				q.Set("limit", strconv.Itoa(limit))
			}
			if offset > 0 {
				q.Set("offset", strconv.Itoa(offset))
			}
			if len(q) > 0 {
				endpoint += "?" + q.Encode()
			}

			resp, err := http.Get(endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			if flagQuiet {
				// In quiet mode, print just the count
				if total, ok := result["total"].(float64); ok {
					fmt.Println(int(total))
				}
				return nil
			}

			return printResult(result)
		},
	}

	cmd.Flags().StringVar(&workflowID, "workflow", "", "Filter by workflow ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of logs to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")

	rootCmd.AddCommand(cmd)
}
