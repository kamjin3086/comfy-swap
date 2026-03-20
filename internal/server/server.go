package server

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"comfy-swap/internal/config"
	"comfy-swap/internal/logs"
	"comfy-swap/internal/proxy"
	"comfy-swap/internal/workflow"

	"github.com/go-chi/chi/v5"
)

type App struct {
	DataDir      string
	SettingsPath string
	Manager      *workflow.Manager
	LogManager   *logs.Manager
	WebFS        embed.FS
	PluginFS     embed.FS
	ProxyFactory func(comfyURL string) *proxy.ComfyProxy
}

func New(dataDir string, webFS embed.FS, pluginFS embed.FS) (*App, error) {
	settingsPath, workflowsDir := config.ResolveDataPaths(dataDir)
	logsDir := config.ResolveLogsDir(dataDir)
	if err := config.EnsureDataDir(dataDir); err != nil {
		return nil, err
	}
	return &App{
		DataDir:      dataDir,
		SettingsPath: settingsPath,
		Manager:      workflow.NewManager(workflowsDir),
		LogManager:   logs.NewManager(logsDir),
		WebFS:        webFS,
		PluginFS:     pluginFS,
		ProxyFactory: proxy.New,
	}, nil
}

func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)

	r.Get("/api/health", a.handleHealth)
	r.Get("/api/settings", a.handleGetSettings)
	r.Put("/api/settings", a.handlePutSettings)
	r.Get("/api/settings/status", a.handleSettingsStatus)

	r.Post("/api/workflows", a.handleCreateWorkflow)
	r.Get("/api/workflows", a.handleListWorkflows)
	r.Get("/api/workflows/{id}", a.handleGetWorkflow)
	r.Put("/api/workflows/{id}", a.handleUpdateWorkflow)
	r.Patch("/api/workflows/{id}/mapping", a.handlePatchMapping)
	r.Delete("/api/workflows/{id}", a.handleDeleteWorkflow)

	r.Post("/api/upload", a.handleUpload)
	r.Post("/api/prompt", a.handlePrompt)
	r.Get("/api/history/{prompt_id}", a.handleHistory)
	r.Get("/api/view", a.handleView)

	r.Get("/api/backup", a.handleBackup)
	r.Post("/api/restore", a.handleRestore)
	r.Post("/api/install-plugin", a.handleInstallPlugin)
	r.Get("/api/download-plugin", a.handleDownloadPlugin)
	r.Get("/api/plugin-status", a.handlePluginStatus)
	r.Post("/api/sync-pending", a.handleSyncPending)

	r.Get("/api/logs", a.handleGetLogs)
	r.Get("/api/logs/{workflow_id}", a.handleGetWorkflowLogs)
	r.Delete("/api/logs/cleanup", a.handleCleanupLogs)

	r.Get("/*", a.handleWeb)
	return r
}

func (a *App) loadSettings() (*config.Settings, error) {
	return config.LoadSettings(a.SettingsPath)
}

func (a *App) loadProxy() (*proxy.ComfyProxy, error) {
	s, err := a.loadSettings()
	if err != nil {
		return nil, err
	}
	if s == nil || strings.TrimSpace(s.ComfyUIURL) == "" {
		return nil, errors.New("comfyui_url is not configured")
	}
	return a.ProxyFactory(strings.TrimRight(s.ComfyUIURL, "/")), nil
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	s, _ := a.loadSettings()
	resp := map[string]interface{}{
		"status":          "ok",
		"workflows_count": 0,
		"version":         "0.1.0",
		"comfyui": map[string]interface{}{
			"reachable": false,
			"url":       "",
		},
	}
	list, err := a.Manager.List()
	if err == nil {
		resp["workflows_count"] = len(list)
	}
	if s != nil {
		respComfy := resp["comfyui"].(map[string]interface{})
		respComfy["url"] = s.ComfyUIURL
		p := a.ProxyFactory(strings.TrimRight(s.ComfyUIURL, "/"))
		if err := p.Health(r.Context()); err == nil {
			respComfy["reachable"] = true
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := a.loadSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if s == nil {
		writeErr(w, http.StatusNotFound, errors.New("settings not initialized"))
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *App) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req config.Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.ComfyUIURL) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("comfyui_url is required"))
		return
	}
	old, _ := a.loadSettings()
	if old != nil {
		req.CreatedAt = old.CreatedAt
	}
	if err := config.SaveSettings(a.SettingsPath, &req); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (a *App) handleSettingsStatus(w http.ResponseWriter, r *http.Request) {
	s, err := a.loadSettings()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"initialized": s != nil,
	})
}

func (a *App) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf workflow.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(wf.ID) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("workflow id is required"))
		return
	}
	if err := a.Manager.SaveNew(&wf); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, wf)
}

func (a *App) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	list, err := a.Manager.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *App) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := a.Manager.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (a *App) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var wf workflow.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	wf.ID = id
	resp, err := a.Manager.Update(&wf)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) handlePatchMapping(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := a.Manager.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	var req struct {
		ParamMapping []workflow.ParamMapping `json:"param_mapping"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	wf.ParamMapping = req.ParamMapping
	resp, err := a.Manager.Update(wf)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.Manager.Delete(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": id})
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	p, err := a.loadProxy()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	f, h, err := r.FormFile("image")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	defer f.Close()
	out, err := p.UploadImage(r.Context(), h.Filename, f)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handlePrompt(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	source := "api"
	if r.Header.Get("X-Comfy-Swap-Source") == "playground" {
		source = "playground"
	}

	p, err := a.loadProxy()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var req workflow.PromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	logEntry := logs.LogEntry{
		WorkflowID: req.WorkflowID,
		Timestamp:  startTime,
		Params:     req.Params,
		Source:     source,
	}

	wf, err := a.Manager.Get(req.WorkflowID)
	if err != nil {
		logEntry.Status = "error"
		logEntry.Error = "workflow not found: " + err.Error()
		logEntry.Duration = time.Since(startTime).Milliseconds()
		_ = a.LogManager.Add(logEntry)
		writeErr(w, http.StatusNotFound, err)
		return
	}
	finalPrompt, warnings, err := a.Manager.ApplyParams(wf, req.Params)
	if err != nil {
		logEntry.Status = "error"
		logEntry.Error = "param apply error: " + err.Error()
		logEntry.Duration = time.Since(startTime).Milliseconds()
		_ = a.LogManager.Add(logEntry)
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	out, err := p.Prompt(r.Context(), finalPrompt)
	if err != nil {
		logEntry.Status = "error"
		logEntry.Error = "comfyui error: " + err.Error()
		logEntry.Duration = time.Since(startTime).Milliseconds()
		_ = a.LogManager.Add(logEntry)
		writeErr(w, http.StatusBadGateway, err)
		return
	}

	logEntry.Status = "success"
	logEntry.Duration = time.Since(startTime).Milliseconds()
	if pid, ok := out["prompt_id"].(string); ok {
		logEntry.PromptID = pid
	}
	_ = a.LogManager.Add(logEntry)

	out["workflow_id"] = req.WorkflowID
	if len(warnings) > 0 {
		out["warnings"] = warnings
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleHistory(w http.ResponseWriter, r *http.Request) {
	p, err := a.loadProxy()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	promptID := chi.URLParam(r, "prompt_id")
	out, err := p.History(r.Context(), promptID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleView(w http.ResponseWriter, r *http.Request) {
	p, err := a.loadProxy()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	resp, err := p.View(r.Context(), r.URL.Query())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()
	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) handleBackup(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.Walk(a.DataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(a.DataDir, path)
		if err != nil {
			return err
		}
		fw, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := zw.Close(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="comfy-swap-backup.zip"`)
	_, _ = w.Write(buf.Bytes())
}

func (a *App) handleRestore(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	for _, f := range zr.File {
		target := filepath.Join(a.DataDir, filepath.Clean(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(a.DataDir)) {
			writeErr(w, http.StatusBadRequest, errors.New("invalid archive path"))
			return
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		rc, err := f.Open()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"restored": true})
}

func (a *App) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomNodesPath string `json:"custom_nodes_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.CustomNodesPath) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("custom_nodes_path is required"))
		return
	}
	installer := &Installer{PluginFS: a.PluginFS}
	path, err := installer.InstallPlugin(req.CustomNodesPath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"installed": true,
		"path":      path,
	})
}

func (a *App) handleDownloadPlugin(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	err := fs.WalkDir(a.PluginFS, "ComfyUI-ComfySwap", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(a.PluginFS, path)
		if err != nil {
			return err
		}
		fw, err := zw.Create(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := zw.Close(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=comfy-swap-plugin.zip")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func (a *App) handlePluginStatus(w http.ResponseWriter, r *http.Request) {
	p, err := a.loadProxy()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "not_configured",
			"installed": false,
			"message":   "ComfyUI URL not configured",
		})
		return
	}

	// Check plugin status endpoint
	ctx := r.Context()
	statusResp, err := p.GetPluginStatus(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "not_installed",
			"installed": false,
			"message":   "Plugin not detected on ComfyUI",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "connected",
		"installed":     true,
		"version":       statusResp["version"],
		"pending_count": statusResp["pending_count"],
	})
}

func (a *App) handleSyncPending(w http.ResponseWriter, r *http.Request) {
	p, err := a.loadProxy()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	ctx := r.Context()
	pendingResp, err := p.GetPendingWorkflows(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}

	workflows, ok := pendingResp["workflows"].([]interface{})
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"synced": 0,
			"errors": []string{},
		})
		return
	}

	synced := 0
	var syncErrors []string

	for _, wfRaw := range workflows {
		wfMap, ok := wfRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Convert to workflow struct
		wfBytes, err := json.Marshal(wfMap)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("marshal error: %v", err))
			continue
		}

		var wf workflow.Workflow
		if err := json.Unmarshal(wfBytes, &wf); err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("unmarshal error: %v", err))
			continue
		}

		// Save workflow
		err = a.Manager.SaveNew(&wf)
		if err != nil {
			// Try update if create fails
			_, err = a.Manager.Update(&wf)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("save %s: %v", wf.ID, err))
				continue
			}
		}

		// Remove from pending queue
		if err := p.RemovePendingWorkflow(ctx, wf.ID); err != nil {
			syncErrors = append(syncErrors, fmt.Sprintf("remove pending %s: %v", wf.ID, err))
		}

		synced++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"synced": synced,
		"errors": syncErrors,
	})
}

func (a *App) handleWeb(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(a.WebFS, "web")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || strings.HasPrefix(path, "api/") {
		path = "index.html"
	}
	f, err := sub.Open(path)
	if err != nil {
		f, err = sub.Open("index.html")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	defer f.Close()
	stat, _ := f.Stat()
	if stat != nil {
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), f.(io.ReadSeeker))
		return
	}
	_, _ = io.Copy(w, f)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]interface{}{
		"error": err.Error(),
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type Installer struct {
	PluginFS embed.FS
}

func (i *Installer) InstallPlugin(targetCustomNodesDir string) (string, error) {
	target := filepath.Join(targetCustomNodesDir, "ComfyUI-ComfySwap")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	sub, err := fs.Sub(i.PluginFS, "ComfyUI-ComfySwap")
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(target, path), 0o755)
		}
		b, err := fs.ReadFile(sub, path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(target, path), b, 0o644)
	})
	if err != nil {
		return "", err
	}
	return target, nil
}

func SaveUploadedFileFromCLI(cli *http.Client, serverURL, filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("image", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	_ = mw.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(serverURL, "/")+"/api/upload", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(b))
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	filename, _ := out["filename"].(string)
	if filename == "" {
		return "", errors.New("upload response missing filename")
	}
	return filename, nil
}

func DownloadOutputToPath(cli *http.Client, serverURL string, q url.Values, savePath string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, strings.TrimRight(serverURL, "/")+"/api/view?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req)
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
	out, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func (a *App) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	query := a.parseLogQuery(r)
	result, err := a.LogManager.Query(query)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleGetWorkflowLogs(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflow_id")
	query := a.parseLogQuery(r)
	query.WorkflowID = workflowID
	result, err := a.LogManager.Query(query)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleCleanupLogs(w http.ResponseWriter, r *http.Request) {
	s, _ := a.loadSettings()
	days := s.GetLogRetentionDays()
	if err := a.LogManager.Cleanup(days); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "cleanup completed",
		"retention_days": days,
	})
}

func (a *App) parseLogQuery(r *http.Request) logs.LogQuery {
	q := logs.LogQuery{}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			q.Limit = n
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil && n >= 0 {
			q.Offset = n
		}
	}
	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			q.StartTime = &t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			q.EndTime = &t
		}
	}

	return q
}
