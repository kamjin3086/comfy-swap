package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

type ComfyProxy struct {
	BaseURL string
	Client  *http.Client
}

func New(baseURL string) *ComfyProxy {
	return &ComfyProxy{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *ComfyProxy) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("comfyui error (%d): %s", resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (p *ComfyProxy) Prompt(ctx context.Context, prompt map[string]interface{}) (map[string]interface{}, error) {
	req := map[string]interface{}{"prompt": prompt}
	out := map[string]interface{}{}
	err := p.doJSON(ctx, http.MethodPost, "/prompt", req, &out)
	return out, err
}

func (p *ComfyProxy) History(ctx context.Context, promptID string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	err := p.doJSON(ctx, http.MethodGet, "/history/"+url.PathEscape(promptID), nil, &out)
	return out, err
}

func (p *ComfyProxy) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.BaseURL+"/system_stats", nil)
	if err != nil {
		return err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("comfyui health failed: %d", resp.StatusCode)
	}
	return nil
}

func (p *ComfyProxy) View(ctx context.Context, query url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.BaseURL+"/view?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return p.Client.Do(req)
}

func (p *ComfyProxy) UploadImage(ctx context.Context, filename string, r io.Reader) (map[string]interface{}, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, r); err != nil {
		return nil, err
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/upload/image", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("comfyui upload error (%d): %s", resp.StatusCode, string(b))
	}
	out := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *ComfyProxy) GetPluginStatus(ctx context.Context) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	err := p.doJSON(ctx, http.MethodGet, "/comfyswap/status", nil, &out)
	return out, err
}

func (p *ComfyProxy) GetPendingWorkflows(ctx context.Context) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	err := p.doJSON(ctx, http.MethodGet, "/comfyswap/pending", nil, &out)
	return out, err
}

func (p *ComfyProxy) RemovePendingWorkflow(ctx context.Context, workflowID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p.BaseURL+"/comfyswap/pending/"+url.PathEscape(workflowID), nil)
	if err != nil {
		return err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove pending error (%d): %s", resp.StatusCode, string(b))
	}
	return nil
}
