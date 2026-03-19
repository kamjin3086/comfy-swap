package logs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type LogEntry struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Status     string                 `json:"status"`
	PromptID   string                 `json:"prompt_id,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   int64                  `json:"duration_ms,omitempty"`
	Source     string                 `json:"source"`
}

type LogQuery struct {
	WorkflowID string
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

type LogResult struct {
	Entries []LogEntry `json:"entries"`
	Total   int        `json:"total"`
	HasMore bool       `json:"has_more"`
}

type Manager struct {
	dir string
	mu  sync.RWMutex
}

func NewManager(logsDir string) *Manager {
	_ = os.MkdirAll(logsDir, 0o755)
	return &Manager{dir: logsDir}
}

func (m *Manager) logFilePath(workflowID string, date time.Time) string {
	dateStr := date.Format("2006-01-02")
	return filepath.Join(m.dir, workflowID, dateStr+".jsonl")
}

func (m *Manager) Add(entry LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if entry.ID == "" {
		entry.ID = generateID()
	}

	wfDir := filepath.Join(m.dir, entry.WorkflowID)
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		return err
	}

	filePath := m.logFilePath(entry.WorkflowID, entry.Timestamp)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

func (m *Manager) Query(q LogQuery) (*LogResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allEntries []LogEntry

	if q.WorkflowID != "" {
		entries, err := m.readWorkflowLogs(q.WorkflowID, q.StartTime, q.EndTime)
		if err != nil {
			return nil, err
		}
		allEntries = entries
	} else {
		wfDirs, err := os.ReadDir(m.dir)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		for _, d := range wfDirs {
			if !d.IsDir() {
				continue
			}
			entries, err := m.readWorkflowLogs(d.Name(), q.StartTime, q.EndTime)
			if err != nil {
				continue
			}
			allEntries = append(allEntries, entries...)
		}
	}

	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.After(allEntries[j].Timestamp)
	})

	total := len(allEntries)
	if q.Offset > 0 && q.Offset < len(allEntries) {
		allEntries = allEntries[q.Offset:]
	} else if q.Offset >= len(allEntries) {
		allEntries = nil
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	hasMore := len(allEntries) > limit
	if len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}

	return &LogResult{
		Entries: allEntries,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

func (m *Manager) readWorkflowLogs(workflowID string, start, end *time.Time) ([]LogEntry, error) {
	wfDir := filepath.Join(m.dir, workflowID)
	files, err := os.ReadDir(wfDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []LogEntry
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
			continue
		}

		dateStr := f.Name()[:len(f.Name())-6]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if start != nil && fileDate.Before(start.Truncate(24*time.Hour)) {
			continue
		}
		if end != nil && fileDate.After(end.Truncate(24*time.Hour)) {
			continue
		}

		fileEntries, err := m.readLogFile(filepath.Join(wfDir, f.Name()))
		if err != nil {
			continue
		}

		for _, e := range fileEntries {
			if start != nil && e.Timestamp.Before(*start) {
				continue
			}
			if end != nil && e.Timestamp.After(*end) {
				continue
			}
			entries = append(entries, e)
		}
	}

	return entries, nil
}

func (m *Manager) readLogFile(path string) ([]LogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e LogEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (m *Manager) Cleanup(retentionDays int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays).Truncate(24 * time.Hour)

	wfDirs, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, d := range wfDirs {
		if !d.IsDir() {
			continue
		}
		wfDir := filepath.Join(m.dir, d.Name())
		files, err := os.ReadDir(wfDir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
				continue
			}
			dateStr := f.Name()[:len(f.Name())-6]
			fileDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			if fileDate.Before(cutoff) {
				_ = os.Remove(filepath.Join(wfDir, f.Name()))
			}
		}

		remaining, _ := os.ReadDir(wfDir)
		if len(remaining) == 0 {
			_ = os.Remove(wfDir)
		}
	}

	return nil
}

func (m *Manager) GetWorkflowStats(workflowID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := m.readWorkflowLogs(workflowID, nil, nil)
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
