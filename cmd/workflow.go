package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage workflow parameters and mappings",
	}

	// workflow params <id> - show params in detail
	paramsCmd := &cobra.Command{
		Use:   "params <workflow_id>",
		Short: "Show workflow parameters in detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showWorkflowParams(args[0])
		},
	}

	// workflow update-param <id> <param_name> [flags]
	var paramName, paramType, paramDefault, paramDesc string
	updateParamCmd := &cobra.Command{
		Use:   "update-param <workflow_id>",
		Short: "Update a parameter's properties",
		Long: `Update properties of an existing parameter.

Examples:
  # Rename a parameter
  comfy-swap workflow update-param my-workflow --name old_name --rename new_name
  
  # Change default value
  comfy-swap workflow update-param my-workflow --name seed --default 42
  
  # Update description
  comfy-swap workflow update-param my-workflow --name prompt --desc "Main generation prompt"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rename, _ := cmd.Flags().GetString("rename")
			return updateParam(args[0], paramName, rename, paramType, paramDefault, paramDesc)
		},
	}
	updateParamCmd.Flags().StringVar(&paramName, "name", "", "Parameter name to update (required)")
	updateParamCmd.Flags().String("rename", "", "New name for the parameter")
	updateParamCmd.Flags().StringVar(&paramType, "type", "", "Parameter type (string, integer, float, boolean, image)")
	updateParamCmd.Flags().StringVar(&paramDefault, "default", "", "Default value")
	updateParamCmd.Flags().StringVar(&paramDesc, "desc", "", "Description")
	updateParamCmd.MarkFlagRequired("name")

	// workflow add-param <id> --name <name> --type <type> --node <node_id> --field <field>
	var nodeID, fieldName string
	addParamCmd := &cobra.Command{
		Use:   "add-param <workflow_id>",
		Short: "Add a new parameter mapping",
		Long: `Add a new parameter that maps to a node field.

Examples:
  comfy-swap workflow add-param my-workflow --name steps --type integer --node 3 --field steps --default 20
  comfy-swap workflow add-param my-workflow --name prompt --type string --node 6 --field text`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return addParam(args[0], paramName, paramType, paramDefault, paramDesc, nodeID, fieldName)
		},
	}
	addParamCmd.Flags().StringVar(&paramName, "name", "", "Parameter name (required)")
	addParamCmd.Flags().StringVar(&paramType, "type", "string", "Parameter type")
	addParamCmd.Flags().StringVar(&paramDefault, "default", "", "Default value")
	addParamCmd.Flags().StringVar(&paramDesc, "desc", "", "Description")
	addParamCmd.Flags().StringVar(&nodeID, "node", "", "Target node ID (required)")
	addParamCmd.Flags().StringVar(&fieldName, "field", "", "Target field name (required)")
	addParamCmd.MarkFlagRequired("name")
	addParamCmd.MarkFlagRequired("node")
	addParamCmd.MarkFlagRequired("field")

	// workflow remove-param <id> --name <name>
	removeParamCmd := &cobra.Command{
		Use:   "remove-param <workflow_id>",
		Short: "Remove a parameter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeParam(args[0], paramName)
		},
	}
	removeParamCmd.Flags().StringVar(&paramName, "name", "", "Parameter name to remove (required)")
	removeParamCmd.MarkFlagRequired("name")

	// workflow add-target <id> --name <param_name> --node <node_id> --field <field>
	addTargetCmd := &cobra.Command{
		Use:   "add-target <workflow_id>",
		Short: "Add another node target to an existing parameter",
		Long: `Add another node target to an existing parameter.
This allows one API parameter to control multiple nodes.

Example:
  # Make 'seed' control both node 3 and node 9
  comfy-swap workflow add-target my-workflow --name seed --node 9 --field seed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return addTarget(args[0], paramName, nodeID, fieldName)
		},
	}
	addTargetCmd.Flags().StringVar(&paramName, "name", "", "Parameter name (required)")
	addTargetCmd.Flags().StringVar(&nodeID, "node", "", "Target node ID (required)")
	addTargetCmd.Flags().StringVar(&fieldName, "field", "", "Target field name (required)")
	addTargetCmd.MarkFlagRequired("name")
	addTargetCmd.MarkFlagRequired("node")
	addTargetCmd.MarkFlagRequired("field")

	// workflow remove-target <id> --name <param_name> --node <node_id>
	removeTargetCmd := &cobra.Command{
		Use:   "remove-target <workflow_id>",
		Short: "Remove a node target from a parameter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeTarget(args[0], paramName, nodeID)
		},
	}
	removeTargetCmd.Flags().StringVar(&paramName, "name", "", "Parameter name (required)")
	removeTargetCmd.Flags().StringVar(&nodeID, "node", "", "Target node ID to remove (required)")
	removeTargetCmd.MarkFlagRequired("name")
	removeTargetCmd.MarkFlagRequired("node")

	// workflow nodes <id> - show all nodes in the workflow
	nodesCmd := &cobra.Command{
		Use:   "nodes <workflow_id>",
		Short: "List all nodes in the workflow",
		Long:  "Show all nodes and their configurable fields. Useful for adding new parameter mappings.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showWorkflowNodes(args[0])
		},
	}

	// workflow delete <id>
	var force bool
	deleteCmd := &cobra.Command{
		Use:   "delete <workflow_id>",
		Short: "Delete a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteWorkflow(args[0], force)
		},
	}
	deleteCmd.Flags().BoolVar(&force, "force", false, "Skip confirmation")

	workflowCmd.AddCommand(paramsCmd)
	workflowCmd.AddCommand(updateParamCmd)
	workflowCmd.AddCommand(addParamCmd)
	workflowCmd.AddCommand(removeParamCmd)
	workflowCmd.AddCommand(addTargetCmd)
	workflowCmd.AddCommand(removeTargetCmd)
	workflowCmd.AddCommand(nodesCmd)
	workflowCmd.AddCommand(deleteCmd)

	rootCmd.AddCommand(workflowCmd)
}

func showWorkflowParams(workflowID string) error {
	resp, err := http.Get(flagServer + "/api/workflows/" + url.PathEscape(workflowID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
	}

	var wf map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		return err
	}

	params, _ := wf["param_mapping"].([]interface{})
	if flagQuiet {
		for _, p := range params {
			pm, _ := p.(map[string]interface{})
			fmt.Println(pm["name"])
		}
		return nil
	}

	return printResult(map[string]interface{}{
		"workflow_id":   workflowID,
		"name":          wf["name"],
		"param_mapping": params,
	})
}

func showWorkflowNodes(workflowID string) error {
	resp, err := http.Get(flagServer + "/api/workflows/" + url.PathEscape(workflowID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
	}

	var wf map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		return err
	}

	comfyWorkflow, _ := wf["comfyui_workflow"].(map[string]interface{})
	if comfyWorkflow == nil {
		return errors.New("no comfyui_workflow found")
	}

	nodes := []map[string]interface{}{}
	for nodeID, nodeData := range comfyWorkflow {
		node, _ := nodeData.(map[string]interface{})
		if node == nil {
			continue
		}

		classType, _ := node["class_type"].(string)
		inputs, _ := node["inputs"].(map[string]interface{})

		fields := []map[string]interface{}{}
		for fieldName, fieldValue := range inputs {
			// Skip array values (node connections)
			if _, isArray := fieldValue.([]interface{}); isArray {
				continue
			}
			fieldType := "string"
			switch v := fieldValue.(type) {
			case float64:
				if v == float64(int(v)) {
					fieldType = "integer"
				} else {
					fieldType = "float"
				}
			case bool:
				fieldType = "boolean"
			}
			fields = append(fields, map[string]interface{}{
				"name":    fieldName,
				"type":    fieldType,
				"current": fieldValue,
			})
		}

		if len(fields) > 0 {
			nodes = append(nodes, map[string]interface{}{
				"node_id":    nodeID,
				"class_type": classType,
				"fields":     fields,
			})
		}
	}

	if flagQuiet {
		for _, n := range nodes {
			fmt.Printf("%s (%s)\n", n["node_id"], n["class_type"])
		}
		return nil
	}

	return printResult(map[string]interface{}{
		"workflow_id": workflowID,
		"nodes":       nodes,
	})
}

func getWorkflowMapping(workflowID string) ([]map[string]interface{}, error) {
	resp, err := http.Get(flagServer + "/api/workflows/" + url.PathEscape(workflowID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
	}

	var wf map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&wf); err != nil {
		return nil, err
	}

	params, _ := wf["param_mapping"].([]interface{})
	result := make([]map[string]interface{}, 0, len(params))
	for _, p := range params {
		pm, _ := p.(map[string]interface{})
		if pm != nil {
			result = append(result, pm)
		}
	}
	return result, nil
}

func saveMapping(workflowID string, mapping []map[string]interface{}) error {
	payload := map[string]interface{}{
		"param_mapping": mapping,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPatch, flagServer+"/api/workflows/"+url.PathEscape(workflowID)+"/mapping", bytes.NewReader(body))
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

	return nil
}

func updateParam(workflowID, paramName, rename, paramType, paramDefault, paramDesc string) error {
	if paramName == "" {
		return errors.New("--name is required")
	}

	mapping, err := getWorkflowMapping(workflowID)
	if err != nil {
		return err
	}

	found := false
	for _, pm := range mapping {
		if pm["name"] == paramName {
			found = true
			if rename != "" {
				pm["name"] = rename
			}
			if paramType != "" {
				pm["type"] = paramType
			}
			if paramDefault != "" {
				pm["default"] = parseValue(paramDefault, pm["type"].(string))
			}
			if paramDesc != "" {
				pm["description"] = paramDesc
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("parameter '%s' not found", paramName)
	}

	if err := saveMapping(workflowID, mapping); err != nil {
		return err
	}

	if flagQuiet {
		fmt.Println("ok")
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "updated",
		"workflow_id": workflowID,
		"param":       paramName,
	})
}

func addParam(workflowID, name, paramType, defaultVal, desc, nodeID, field string) error {
	mapping, err := getWorkflowMapping(workflowID)
	if err != nil {
		return err
	}

	// Check if param already exists
	for _, pm := range mapping {
		if pm["name"] == name {
			return fmt.Errorf("parameter '%s' already exists", name)
		}
	}

	newParam := map[string]interface{}{
		"name":        name,
		"type":        paramType,
		"default":     parseValue(defaultVal, paramType),
		"description": desc,
		"targets": []map[string]string{
			{"node_id": nodeID, "field": field},
		},
	}

	mapping = append(mapping, newParam)

	if err := saveMapping(workflowID, mapping); err != nil {
		return err
	}

	if flagQuiet {
		fmt.Println(name)
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "added",
		"workflow_id": workflowID,
		"param":       name,
	})
}

func removeParam(workflowID, name string) error {
	mapping, err := getWorkflowMapping(workflowID)
	if err != nil {
		return err
	}

	newMapping := make([]map[string]interface{}, 0, len(mapping))
	found := false
	for _, pm := range mapping {
		if pm["name"] == name {
			found = true
			continue
		}
		newMapping = append(newMapping, pm)
	}

	if !found {
		return fmt.Errorf("parameter '%s' not found", name)
	}

	if err := saveMapping(workflowID, newMapping); err != nil {
		return err
	}

	if flagQuiet {
		fmt.Println("ok")
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "removed",
		"workflow_id": workflowID,
		"param":       name,
	})
}

func addTarget(workflowID, paramName, nodeID, field string) error {
	mapping, err := getWorkflowMapping(workflowID)
	if err != nil {
		return err
	}

	found := false
	for _, pm := range mapping {
		if pm["name"] == paramName {
			found = true
			targets, _ := pm["targets"].([]interface{})
			newTargets := make([]map[string]string, 0, len(targets)+1)
			for _, t := range targets {
				tm, _ := t.(map[string]interface{})
				if tm != nil {
					newTargets = append(newTargets, map[string]string{
						"node_id": fmt.Sprintf("%v", tm["node_id"]),
						"field":   fmt.Sprintf("%v", tm["field"]),
					})
				}
			}
			newTargets = append(newTargets, map[string]string{
				"node_id": nodeID,
				"field":   field,
			})
			pm["targets"] = newTargets
			break
		}
	}

	if !found {
		return fmt.Errorf("parameter '%s' not found", paramName)
	}

	if err := saveMapping(workflowID, mapping); err != nil {
		return err
	}

	if flagQuiet {
		fmt.Println("ok")
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "target_added",
		"workflow_id": workflowID,
		"param":       paramName,
		"node_id":     nodeID,
	})
}

func removeTarget(workflowID, paramName, nodeID string) error {
	mapping, err := getWorkflowMapping(workflowID)
	if err != nil {
		return err
	}

	found := false
	for _, pm := range mapping {
		if pm["name"] == paramName {
			found = true
			targets, _ := pm["targets"].([]interface{})
			newTargets := make([]map[string]string, 0, len(targets))
			for _, t := range targets {
				tm, _ := t.(map[string]interface{})
				if tm != nil && fmt.Sprintf("%v", tm["node_id"]) != nodeID {
					newTargets = append(newTargets, map[string]string{
						"node_id": fmt.Sprintf("%v", tm["node_id"]),
						"field":   fmt.Sprintf("%v", tm["field"]),
					})
				}
			}
			pm["targets"] = newTargets
			break
		}
	}

	if !found {
		return fmt.Errorf("parameter '%s' not found", paramName)
	}

	if err := saveMapping(workflowID, mapping); err != nil {
		return err
	}

	if flagQuiet {
		fmt.Println("ok")
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "target_removed",
		"workflow_id": workflowID,
		"param":       paramName,
		"node_id":     nodeID,
	})
}

func deleteWorkflow(workflowID string, force bool) error {
	if !force {
		fmt.Printf("Delete workflow '%s'? [y/N]: ", workflowID)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	req, err := http.NewRequest(http.MethodDelete, flagServer+"/api/workflows/"+url.PathEscape(workflowID), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
	}

	if flagQuiet {
		fmt.Println("ok")
		return nil
	}

	return printResult(map[string]interface{}{
		"status":      "deleted",
		"workflow_id": workflowID,
	})
}

func parseValue(value, paramType string) interface{} {
	switch paramType {
	case "integer":
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	case "float":
		var f float64
		fmt.Sscanf(value, "%f", &f)
		return f
	case "boolean":
		return strings.ToLower(value) == "true"
	default:
		return value
	}
}

