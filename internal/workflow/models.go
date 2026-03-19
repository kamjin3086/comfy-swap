package workflow

import "time"

type ParamType string

const (
	ParamString  ParamType = "string"
	ParamInteger ParamType = "integer"
	ParamFloat   ParamType = "float"
	ParamBoolean ParamType = "boolean"
	ParamImage   ParamType = "image"
)

type MappingTarget struct {
	NodeID string `json:"node_id"`
	Field  string `json:"field"`
}

type ParamMapping struct {
	Name        string          `json:"name"`
	Type        ParamType       `json:"type"`
	Default     interface{}     `json:"default"`
	Description string          `json:"description,omitempty"`
	Targets     []MappingTarget `json:"targets"`
}

type Workflow struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Version         int                    `json:"version"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	ComfyUIWorkflow map[string]interface{} `json:"comfyui_workflow"`
	ParamMapping    []ParamMapping         `json:"param_mapping"`
}

type WorkflowSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ParamsCount int       `json:"params_count"`
	Version     int       `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateChangeSummary struct {
	ParamsAdded          []string `json:"params_added"`
	ParamsRemoved        []string `json:"params_removed"`
	ParamsUnchanged      []string `json:"params_unchanged"`
	ParamsTargetsChanged []string `json:"params_targets_changed"`
}

type UpdateWorkflowResponse struct {
	Status  string              `json:"status"`
	Changes UpdateChangeSummary `json:"changes"`
}

type PromptRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Params     map[string]interface{} `json:"params"`
}
