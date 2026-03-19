package workflow

import (
	"path/filepath"
	"testing"
)

func TestApplyParamsWithTypeCoercionAndMultiTarget(t *testing.T) {
	t.Parallel()
	m := NewManager(filepath.Join(t.TempDir(), "workflows"))
	wf := &Workflow{
		ID:   "test",
		Name: "test",
		ComfyUIWorkflow: map[string]interface{}{
			"1": map[string]interface{}{
				"class_type": "KSampler",
				"inputs": map[string]interface{}{
					"seed":    0,
					"steps":   20,
					"denoise": 1.0,
				},
			},
			"2": map[string]interface{}{
				"class_type": "KSampler",
				"inputs": map[string]interface{}{
					"seed":    0,
					"denoise": 1.0,
				},
			},
		},
		ParamMapping: []ParamMapping{
			{
				Name:    "seed",
				Type:    ParamInteger,
				Default: 0,
				Targets: []MappingTarget{
					{NodeID: "1", Field: "seed"},
					{NodeID: "2", Field: "seed"},
				},
			},
			{
				Name:    "denoise",
				Type:    ParamFloat,
				Default: 0.8,
				Targets: []MappingTarget{
					{NodeID: "1", Field: "denoise"},
					{NodeID: "2", Field: "denoise"},
				},
			},
			{
				Name:    "enabled",
				Type:    ParamBoolean,
				Default: false,
				Targets: []MappingTarget{
					{NodeID: "1", Field: "enabled"},
				},
			},
		},
	}
	params := map[string]interface{}{
		"seed":    "42",
		"denoise": "0.65",
		"enabled": "true",
	}

	prompt, warnings, err := m.ApplyParams(wf, params)
	if err != nil {
		t.Fatalf("ApplyParams returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", warnings)
	}

	node1 := prompt["1"].(map[string]interface{})
	node2 := prompt["2"].(map[string]interface{})
	in1 := node1["inputs"].(map[string]interface{})
	in2 := node2["inputs"].(map[string]interface{})

	if got := asInt(t, in1["seed"]); got != 42 {
		t.Fatalf("node1 seed: got %v, want 42", got)
	}
	if got := asInt(t, in2["seed"]); got != 42 {
		t.Fatalf("node2 seed: got %v, want 42", got)
	}
	if got := in1["denoise"].(float64); got != 0.65 {
		t.Fatalf("node1 denoise: got %v, want 0.65", got)
	}
	if got := in2["denoise"].(float64); got != 0.65 {
		t.Fatalf("node2 denoise: got %v, want 0.65", got)
	}
	if got := in1["enabled"].(bool); !got {
		t.Fatalf("node1 enabled: got %v, want true", got)
	}
}

func TestUpdateReturnsChangeSummary(t *testing.T) {
	t.Parallel()
	workDir := filepath.Join(t.TempDir(), "workflows")
	m := NewManager(workDir)
	orig := &Workflow{
		ID:   "wf1",
		Name: "workflow-1",
		ComfyUIWorkflow: map[string]interface{}{
			"1": map[string]interface{}{"inputs": map[string]interface{}{"seed": 0}},
		},
		ParamMapping: []ParamMapping{
			{Name: "seed", Type: ParamInteger, Targets: []MappingTarget{{NodeID: "1", Field: "seed"}}},
			{Name: "steps", Type: ParamInteger, Targets: []MappingTarget{{NodeID: "1", Field: "steps"}}},
		},
	}
	if err := m.SaveNew(orig); err != nil {
		t.Fatalf("SaveNew failed: %v", err)
	}

	updated := &Workflow{
		ID:   "wf1",
		Name: "workflow-1",
		ComfyUIWorkflow: map[string]interface{}{
			"1": map[string]interface{}{"inputs": map[string]interface{}{"seed": 0}},
			"2": map[string]interface{}{"inputs": map[string]interface{}{"seed": 0}},
		},
		ParamMapping: []ParamMapping{
			{Name: "seed", Type: ParamInteger, Targets: []MappingTarget{{NodeID: "1", Field: "seed"}, {NodeID: "2", Field: "seed"}}},
			{Name: "cfg", Type: ParamFloat, Targets: []MappingTarget{{NodeID: "1", Field: "cfg"}}},
		},
	}
	resp, err := m.Update(updated)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	assertContains(t, resp.Changes.ParamsAdded, "cfg")
	assertContains(t, resp.Changes.ParamsRemoved, "steps")
	assertContains(t, resp.Changes.ParamsTargetsChanged, "seed")
}

func assertContains(t *testing.T, list []string, want string) {
	t.Helper()
	for _, x := range list {
		if x == want {
			return
		}
	}
	t.Fatalf("expected %q in list %v", want, list)
}

func asInt(t *testing.T, v interface{}) int {
	t.Helper()
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		t.Fatalf("unexpected numeric type: %T", v)
		return 0
	}
}
