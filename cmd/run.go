package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"comfy-swap/internal/client"
	"comfy-swap/internal/workflow"

	"github.com/spf13/cobra"
)

func init() {
	var paramsJSON string
	var wait bool
	var timeoutSec int
	var savePath string
	cmd := &cobra.Command{
		Use:   "run <workflow_id> [key=value...]",
		Short: "Run a workflow",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("workflow_id is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]
			c := client.New(flagServer)
			wf, err := c.GetWorkflow(workflowID)
			if err != nil {
				return err
			}
			typeMap := map[string]workflow.ParamType{}
			for _, pm := range wf.ParamMapping {
				typeMap[pm.Name] = pm.Type
			}

			params, err := parseParams(paramsJSON, args[1:])
			if err != nil {
				return err
			}
			if err := resolveAtFileParams(c, params, typeMap); err != nil {
				return err
			}

			runResp, err := c.RunPrompt(workflowID, params)
			if err != nil {
				return err
			}
			if !wait {
				if flagQuiet {
					promptID, _ := runResp["prompt_id"].(string)
					return printResult(promptID)
				}
				return printResult(runResp)
			}

			promptID, _ := runResp["prompt_id"].(string)
			if promptID == "" {
				return errors.New("run response missing prompt_id")
			}
			h, err := c.WaitForCompletion(promptID, time.Duration(timeoutSec)*time.Second)
			if err != nil {
				return err
			}
			outputs := extractOutputs(h, promptID)
			if savePath == "" {
				result := map[string]interface{}{
					"prompt_id": promptID,
					"status":    "completed",
					"outputs":   outputs,
				}
				return printResult(result)
			}
			files, err := downloadOutputs(c, outputs, savePath)
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
				"prompt_id": promptID,
				"status":    "completed",
				"files":     files,
			})
		},
	}
	cmd.Flags().StringVar(&paramsJSON, "params", "", "JSON params string or '-' for stdin")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for completion")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 300, "wait timeout in seconds")
	cmd.Flags().StringVar(&savePath, "save", "", "save output to directory or file path")
	rootCmd.AddCommand(cmd)
}

func parseParams(paramsJSON string, kvs []string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	if paramsJSON != "" {
		var raw string
		if paramsJSON == "-" {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, err
			}
			raw = string(b)
		} else {
			raw = paramsJSON
		}
		if err := jsonUnmarshalString(raw, &out); err != nil {
			return nil, err
		}
	}
	for _, kv := range kvs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid param format: %s, expected key=value", kv)
		}
		out[parts[0]] = parts[1]
	}
	return out, nil
}

func resolveAtFileParams(c *client.Client, params map[string]interface{}, typeMap map[string]workflow.ParamType) error {
	for k, v := range params {
		s, ok := v.(string)
		if !ok || !strings.HasPrefix(s, "@") || len(s) < 2 {
			continue
		}
		path := strings.TrimPrefix(s, "@")
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read @file for param %s failed: %w", k, err)
		}
		if typeMap[k] == workflow.ParamImage {
			uploadResp, err := c.UploadImage(path)
			if err != nil {
				return fmt.Errorf("upload image for param %s failed: %w", k, err)
			}
			filename, _ := uploadResp["filename"].(string)
			if filename == "" {
				return fmt.Errorf("upload image for param %s missing filename", k)
			}
			params[k] = filename
			continue
		}
		params[k] = string(b)
	}
	return nil
}

func extractOutputs(history map[string]interface{}, promptID string) []map[string]string {
	item, ok := history[promptID].(map[string]interface{})
	if !ok {
		return nil
	}
	outNode, ok := item["outputs"].(map[string]interface{})
	if !ok {
		return nil
	}
	out := []map[string]string{}
	for _, rawNode := range outNode {
		nodeMap, ok := rawNode.(map[string]interface{})
		if !ok {
			continue
		}
		images, ok := nodeMap["images"].([]interface{})
		if !ok {
			continue
		}
		for _, rawImage := range images {
			im, ok := rawImage.(map[string]interface{})
			if !ok {
				continue
			}
			entry := map[string]string{
				"filename":  asString(im["filename"]),
				"subfolder": asString(im["subfolder"]),
				"type":      asString(im["type"]),
			}
			if entry["type"] == "" {
				entry["type"] = "output"
			}
			out = append(out, entry)
		}
	}
	return out
}

func downloadOutputs(c *client.Client, outputs []map[string]string, savePath string) ([]string, error) {
	if len(outputs) == 0 {
		return nil, errors.New("no output images found")
	}
	isDir := strings.HasSuffix(savePath, "/") || strings.HasSuffix(savePath, "\\")
	if st, err := os.Stat(savePath); err == nil && st.IsDir() {
		isDir = true
	}
	files := []string{}
	for i, o := range outputs {
		filename := o["filename"]
		subfolder := o["subfolder"]
		outType := o["type"]
		target := savePath
		if isDir {
			target = filepath.Join(savePath, filename)
		} else if len(outputs) > 1 {
			ext := filepath.Ext(savePath)
			base := strings.TrimSuffix(savePath, ext)
			target = fmt.Sprintf("%s_%d%s", base, i+1, ext)
		}
		if err := c.DownloadOutput(filename, subfolder, outType, target); err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	return files, nil
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func jsonUnmarshalString(s string, out interface{}) error {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	return dec.Decode(out)
}
