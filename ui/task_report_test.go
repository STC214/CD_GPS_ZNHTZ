package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"comic_downloader_go_playwright_stealth/tasks"
)

func TestCleanupTaskLogRemovesSuccessLog(t *testing.T) {
	runtimeRoot := t.TempDir()
	item := TodoItem{
		ID:         "todo-123",
		Status:     TodoStatusCompleted,
		CreatedAt:  time.Now().UTC(),
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
		Request: tasks.BrowserLaunchRequest{
			URL:         "https://example.com",
			RuntimeRoot: runtimeRoot,
			OutputDir:   filepath.Join(runtimeRoot, "output"),
		},
		Result: tasks.BrowserRunResult{
			Title:         "Example",
			URL:           "https://example.com",
			DownloadedDir: filepath.Join(runtimeRoot, "output"),
		},
	}

	if err := SaveTaskReport(runtimeRoot, item); err != nil {
		t.Fatalf("SaveTaskReport() error = %v", err)
	}
	reportPath := filepath.Join(runtimeRoot, "tasks", "task-todo-123", "report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("ReadFile(report) error = %v", err)
	}
	var report tasks.TaskReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal report error = %v", err)
	}
	if filepath.IsAbs(report.OutputRoot) || filepath.IsAbs(report.ThumbnailPath) || filepath.IsAbs(report.ReportPath) || filepath.IsAbs(report.LogPath) {
		t.Fatalf("report paths should be relative: %+v", report)
	}
	paths := filepath.Join(runtimeRoot, "logs")
	logPath := filepath.Join(paths, "task-todo_123.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log to exist before cleanup: %v", err)
	}
	if err := CleanupTaskLog(runtimeRoot, item.ID); err != nil {
		t.Fatalf("CleanupTaskLog() error = %v", err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log still exists after cleanup: %v", err)
	}
}
