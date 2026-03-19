package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	workflowsDir string
}

func NewManager(workflowsDir string) *Manager {
	return &Manager{workflowsDir: workflowsDir}
}

func (m *Manager) workflowPath(id string) string {
	return filepath.Join(m.workflowsDir, id+".json")
}

func (m *Manager) SaveNew(wf *Workflow) error {
	if wf.ID == "" {
		return errors.New("workflow id is required")
	}
	if _, err := m.Get(wf.ID); err == nil {
		return fmt.Errorf("workflow %q already exists", wf.ID)
	}
	now := time.Now().UTC()
	wf.CreatedAt = now
	wf.UpdatedAt = now
	if wf.Version == 0 {
		wf.Version = 1
	}
	return m.save(wf)
}

func (m *Manager) Update(wf *Workflow) (*UpdateWorkflowResponse, error) {
	old, err := m.Get(wf.ID)
	if err != nil {
		return nil, err
	}
	resp := compareChanges(old, wf)
	wf.CreatedAt = old.CreatedAt
	wf.Version = old.Version + 1
	wf.UpdatedAt = time.Now().UTC()
	if err := m.save(wf); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Manager) save(wf *Workflow) error {
	if err := os.MkdirAll(m.workflowsDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.workflowPath(wf.ID), b, 0o644)
}

func (m *Manager) Get(id string) (*Workflow, error) {
	b, err := os.ReadFile(m.workflowPath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("workflow %q not found", id)
		}
		return nil, err
	}
	var wf Workflow
	if err := json.Unmarshal(b, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func (m *Manager) List() ([]WorkflowSummary, error) {
	if err := os.MkdirAll(m.workflowsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.workflowsDir)
	if err != nil {
		return nil, err
	}
	out := make([]WorkflowSummary, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(m.workflowsDir, e.Name()))
		if err != nil {
			return nil, err
		}
		var wf Workflow
		if err := json.Unmarshal(b, &wf); err != nil {
			return nil, err
		}
		out = append(out, WorkflowSummary{
			ID:          wf.ID,
			Name:        wf.Name,
			ParamsCount: len(wf.ParamMapping),
			Version:     wf.Version,
			UpdatedAt:   wf.UpdatedAt,
		})
	}
	slices.SortFunc(out, func(a, b WorkflowSummary) int {
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})
	return out, nil
}

func (m *Manager) Delete(id string) error {
	err := os.Remove(m.workflowPath(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Manager) ApplyParams(wf *Workflow, params map[string]interface{}) (map[string]interface{}, []string, error) {
	merged := map[string]interface{}{}
	for k, v := range params {
		merged[k] = v
	}
	workflowCopy, err := deepCopyMap(wf.ComfyUIWorkflow)
	if err != nil {
		return nil, nil, err
	}

	warnings := []string{}
	known := map[string]ParamMapping{}
	for _, pm := range wf.ParamMapping {
		known[pm.Name] = pm
	}
	for k := range params {
		if _, ok := known[k]; !ok {
			warnings = append(warnings, fmt.Sprintf("unknown param ignored: %s", k))
		}
	}

	for _, pm := range wf.ParamMapping {
		val, ok := merged[pm.Name]
		if !ok {
			val = pm.Default
		}
		typed, err := coerceType(pm.Type, val, pm.Name)
		if err != nil {
			return nil, nil, err
		}
		for _, t := range pm.Targets {
			rawNode, ok := workflowCopy[t.NodeID]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("target node missing: %s.%s", t.NodeID, t.Field))
				continue
			}
			node, ok := rawNode.(map[string]interface{})
			if !ok {
				warnings = append(warnings, fmt.Sprintf("invalid node format: %s", t.NodeID))
				continue
			}
			inputs, ok := node["inputs"].(map[string]interface{})
			if !ok {
				warnings = append(warnings, fmt.Sprintf("missing inputs: %s", t.NodeID))
				continue
			}
			inputs[t.Field] = typed
		}
	}
	return workflowCopy, warnings, nil
}

func deepCopyMap(src map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst map[string]interface{}
	if err := json.Unmarshal(b, &dst); err != nil {
		return nil, err
	}
	return dst, nil
}

func coerceType(pt ParamType, value interface{}, name string) (interface{}, error) {
	switch pt {
	case ParamString, ParamImage:
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	case ParamInteger:
		switch v := value.(type) {
		case float64:
			return int(v), nil
		case int:
			return v, nil
		case int64:
			return int(v), nil
		case string:
			i, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("%q cannot be converted to integer for param %q", v, name)
			}
			return i, nil
		default:
			return nil, fmt.Errorf("invalid integer type for param %q", name)
		}
	case ParamFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return nil, fmt.Errorf("%q cannot be converted to float for param %q", v, name)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("invalid float type for param %q", name)
		}
	case ParamBoolean:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("%q cannot be converted to boolean for param %q", v, name)
			}
			return b, nil
		default:
			return nil, fmt.Errorf("invalid boolean type for param %q", name)
		}
	default:
		return value, nil
	}
}

func compareChanges(oldWf, newWf *Workflow) UpdateWorkflowResponse {
	resp := UpdateWorkflowResponse{
		Status: "updated",
		Changes: UpdateChangeSummary{
			ParamsAdded:          []string{},
			ParamsRemoved:        []string{},
			ParamsUnchanged:      []string{},
			ParamsTargetsChanged: []string{},
		},
	}
	oldMap := make(map[string]ParamMapping)
	newMap := make(map[string]ParamMapping)
	for _, p := range oldWf.ParamMapping {
		oldMap[p.Name] = p
	}
	for _, p := range newWf.ParamMapping {
		newMap[p.Name] = p
	}

	for name, oldP := range oldMap {
		newP, ok := newMap[name]
		if !ok {
			resp.Changes.ParamsRemoved = append(resp.Changes.ParamsRemoved, name)
			continue
		}
		if sameTargets(oldP.Targets, newP.Targets) {
			resp.Changes.ParamsUnchanged = append(resp.Changes.ParamsUnchanged, name)
		} else {
			resp.Changes.ParamsTargetsChanged = append(resp.Changes.ParamsTargetsChanged, name)
		}
	}
	for name := range newMap {
		if _, ok := oldMap[name]; !ok {
			resp.Changes.ParamsAdded = append(resp.Changes.ParamsAdded, name)
		}
	}
	slices.Sort(resp.Changes.ParamsAdded)
	slices.Sort(resp.Changes.ParamsRemoved)
	slices.Sort(resp.Changes.ParamsUnchanged)
	slices.Sort(resp.Changes.ParamsTargetsChanged)
	return resp
}

func sameTargets(a, b []MappingTarget) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(t MappingTarget) string {
		return t.NodeID + ":" + t.Field
	}
	x := make([]string, 0, len(a))
	y := make([]string, 0, len(b))
	for _, t := range a {
		x = append(x, key(t))
	}
	for _, t := range b {
		y = append(y, key(t))
	}
	slices.Sort(x)
	slices.Sort(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
