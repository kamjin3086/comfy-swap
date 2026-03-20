package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	var name string
	var syncPending bool

	cmd := &cobra.Command{
		Use:   "import [file.json]",
		Short: "Import workflow from JSON file or sync pending from ComfyUI",
		Long: `Import workflows into Comfy-Swap.

Methods:
1. From JSON file: comfy-swap import workflow.json
2. From stdin: cat workflow.json | comfy-swap import -
3. Sync pending from ComfyUI: comfy-swap import --sync

The --sync flag imports workflows that were exported from ComfyUI 
using the "Export to ComfySwap" menu but not yet synced.

Examples:
  comfy-swap import my-workflow.json
  comfy-swap import --sync
  cat workflow.json | comfy-swap import - --name my-workflow`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if syncPending {
				return syncFromComfyUI()
			}
			if len(args) == 0 {
				return errors.New("provide a JSON file path, use '-' for stdin, or use --sync")
			}
			return importFromFile(args[0], name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Override workflow name")
	cmd.Flags().BoolVar(&syncPending, "sync", false, "Sync pending workflows from ComfyUI plugin")

	rootCmd.AddCommand(cmd)
}

func syncFromComfyUI() error {
	// Call the sync-pending endpoint
	resp, err := http.Post(flagServer+"/api/sync-pending", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync failed (%d): %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if flagQuiet {
		synced, _ := result["synced"].(float64)
		fmt.Println(int(synced))
		return nil
	}

	return printResult(result)
}

func importFromFile(filePath, overrideName string) error {
	var fileContent []byte
	var err error

	if filePath == "-" {
		fileContent, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		fileContent, err = os.ReadFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var workflow map[string]interface{}
	if err := json.Unmarshal(fileContent, &workflow); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Override name if provided
	if overrideName != "" {
		workflow["name"] = overrideName
		workflow["id"] = slugify(overrideName)
		fileContent, _ = json.Marshal(workflow)
	}

	// Validate required fields
	if workflow["id"] == nil || workflow["name"] == nil {
		return errors.New("workflow JSON must have 'id' and 'name' fields")
	}

	workflowID, _ := workflow["id"].(string)

	// Try POST first (create)
	resp, err := http.Post(flagServer+"/api/workflows", "application/json", bytes.NewReader(fileContent))
	if err != nil {
		return fmt.Errorf("failed to send to server: %w", err)
	}
	defer resp.Body.Close()

	// If 409 conflict, try PUT (update)
	if resp.StatusCode == 409 {
		req, _ := http.NewRequest(http.MethodPut, flagServer+"/api/workflows/"+workflowID, bytes.NewReader(fileContent))
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(b))
	}

	if flagQuiet {
		fmt.Println(workflowID)
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "imported",
		"workflow_id": workflowID,
		"name":        workflow["name"],
	})
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	value = reg.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 64 {
		value = value[:64]
	}
	if value == "" {
		return "workflow"
	}
	return value
}
