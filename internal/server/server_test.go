package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"comfy-swap/internal/assets"
	"comfy-swap/internal/workflow"
)

func TestAPISmoke(t *testing.T) {
	t.Parallel()
	dataDir := filepath.Join(t.TempDir(), "data")
	app, err := New(dataDir, assets.WebFS, assets.PluginFS)
	if err != nil {
		t.Fatalf("New app failed: %v", err)
	}
	r := app.Router()

	// settings status -> not initialized
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/settings/status", nil)
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("settings/status code: got %d", rec.Code)
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if out["initialized"] != false {
			t.Fatalf("expected initialized=false, got %v", out["initialized"])
		}
	}

	// put settings
	{
		body := []byte(`{"comfyui_url":"http://127.0.0.1:8188"}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("put settings code: got %d, body=%s", rec.Code, rec.Body.String())
		}
	}

	// create workflow
	wf := workflow.Workflow{
		ID:   "txt2img",
		Name: "txt2img",
		ComfyUIWorkflow: map[string]interface{}{
			"1": map[string]interface{}{
				"class_type": "CLIPTextEncode",
				"inputs": map[string]interface{}{
					"text": "",
				},
			},
		},
		ParamMapping: []workflow.ParamMapping{
			{
				Name:    "prompt",
				Type:    workflow.ParamString,
				Default: "",
				Targets: []workflow.MappingTarget{{NodeID: "1", Field: "text"}},
			},
		},
	}
	b, _ := json.Marshal(wf)
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader(b))
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create workflow code: got %d, body=%s", rec.Code, rec.Body.String())
		}
	}

	// list workflows
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list workflows code: got %d", rec.Code)
		}
		var list []workflow.WorkflowSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal list failed: %v", err)
		}
		if len(list) != 1 || list[0].ID != "txt2img" {
			t.Fatalf("unexpected workflow list: %+v", list)
		}
	}

	// patch mapping
	{
		body := []byte(`{"param_mapping":[{"name":"prompt","type":"string","default":"hello","description":"positive prompt","targets":[{"node_id":"1","field":"text"}]}]}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/workflows/txt2img/mapping", bytes.NewReader(body))
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch mapping code: got %d, body=%s", rec.Code, rec.Body.String())
		}
	}
}
