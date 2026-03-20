package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "View or update server configuration",
	}

	// config get
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(flagServer + "/api/settings")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return errors.New("settings not initialized - run 'comfy-swap config set --comfyui-url <url>' first")
			}
			if resp.StatusCode >= 400 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
			}

			var settings map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
				return err
			}

			if flagQuiet {
				if url, ok := settings["comfyui_url"].(string); ok {
					fmt.Println(url)
				}
				return nil
			}
			return printResult(settings)
		},
	}

	// config set
	var comfyuiURL string
	var logRetentionDays int
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Update configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if comfyuiURL == "" {
				return errors.New("--comfyui-url is required")
			}

			payload := map[string]interface{}{
				"comfyui_url": comfyuiURL,
			}
			if logRetentionDays > 0 {
				payload["log_retention_days"] = logRetentionDays
			}

			body, _ := json.Marshal(payload)
			req, err := http.NewRequest(http.MethodPut, flagServer+"/api/settings", strings.NewReader(string(body)))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				b, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
			}

			var settings map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
				return err
			}

			if flagQuiet {
				fmt.Println("ok")
				return nil
			}
			return printResult(settings)
		},
	}
	setCmd.Flags().StringVar(&comfyuiURL, "comfyui-url", "", "ComfyUI server URL (required)")
	setCmd.Flags().IntVar(&logRetentionDays, "log-retention-days", 0, "Log retention days (optional)")

	configCmd.AddCommand(getCmd)
	configCmd.AddCommand(setCmd)
	rootCmd.AddCommand(configCmd)
}
