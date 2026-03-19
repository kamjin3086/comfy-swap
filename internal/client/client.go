package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"comfy-swap/internal/workflow"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type HealthResult struct {
	Status         string                 `json:"status"`
	ComfyUI        map[string]interface{} `json:"comfyui"`
	WorkflowsCount int                    `json:"workflows_count"`
	Version        string                 `json:"version"`
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) Health() (*HealthResult, error) {
	var out HealthResult
	if err := c.doJSON(context.Background(), http.MethodGet, "/api/health", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListWorkflows() ([]workflow.WorkflowSummary, error) {
	var out []workflow.WorkflowSummary
	if err := c.doJSON(context.Background(), http.MethodGet, "/api/workflows", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetWorkflow(id string) (*workflow.Workflow, error) {
	var out workflow.Workflow
	if err := c.doJSON(context.Background(), http.MethodGet, "/api/workflows/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RunPrompt(workflowID string, params map[string]interface{}) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := c.doJSON(context.Background(), http.MethodPost, "/api/prompt", map[string]interface{}{
		"workflow_id": workflowID,
		"params":      params,
	}, &out)
	return out, err
}

func (c *Client) GetHistory(promptID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := c.doJSON(context.Background(), http.MethodGet, "/api/history/"+url.PathEscape(promptID), nil, &out)
	return out, err
}

func (c *Client) WaitForCompletion(promptID string, timeout time.Duration) (map[string]interface{}, error) {
	start := time.Now()
	interval := time.Second
	for {
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("timeout after %s", timeout)
		}
		h, err := c.GetHistory(promptID)
		if err != nil {
			return nil, err
		}
		if _, ok := h[promptID]; ok {
			return h, nil
		}
		time.Sleep(interval)
		if interval < 10*time.Second {
			interval *= 2
		}
	}
}

func (c *Client) DownloadOutput(filename, subfolder, outputType, savePath string) error {
	q := url.Values{}
	q.Set("filename", filename)
	if subfolder != "" {
		q.Set("subfolder", subfolder)
	}
	if outputType != "" {
		q.Set("type", outputType)
	} else {
		q.Set("type", "output")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.BaseURL+"/api/view?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, string(b))
	}
	if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func (c *Client) UploadImage(filePath string) (map[string]interface{}, error) {
	// Reuse api/upload endpoint with raw multipart manually.
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return uploadMultipart(c.HTTPClient, c.BaseURL+"/api/upload", f, filepath.Base(filePath))
}

func uploadMultipart(httpClient *http.Client, endpoint string, file io.Reader, filename string) (map[string]interface{}, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, file); err != nil {
		return nil, err
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(b))
	}
	out := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
