package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/siteflow"
	"comic_downloader_go_playwright_stealth/tasks"
)

func TestLoadTaskViewReadsReport(t *testing.T) {
	workdir := t.TempDir()
	reportPath := filepath.Join(workdir, "report.json")
	report := tasks.TaskReport{
		TaskID: "1",
		Manifest: siteflow.TaskManifestSummary{
			Site:       "zeri",
			Title:      "Sample Title",
			PrimaryURL: "https://example.com/item",
			AssetCount: 3,
			Blocked:    false,
		},
		State:         tasks.TaskStateCompleted,
		Verification:  "committed",
		OutputRoot:    "downloads/task-1",
		ThumbnailRoot: "thumbnails/task-1",
		ThumbnailPath: "thumbnails/task-1/thumb.jpg",
		StatePath:     "state.json",
		ReportPath:    "report.json",
		LogPath:       "task.log",
		CreatedAt:     time.Now().UTC(),
		StartedAt:     time.Now().UTC(),
		FinishedAt:    time.Now().UTC(),
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	view, err := LoadTaskView(reportPath)
	if err != nil {
		t.Fatalf("LoadTaskView() error = %v", err)
	}
	if view.Title != "Sample Title" {
		t.Fatalf("Title = %q", view.Title)
	}
	if view.Site != "zeri" {
		t.Fatalf("Site = %q", view.Site)
	}
	if view.State != string(tasks.TaskStateCompleted) {
		t.Fatalf("State = %q", view.State)
	}
	if view.AssetCount != 3 {
		t.Fatalf("AssetCount = %d", view.AssetCount)
	}
	if view.OutputRoot != filepath.Clean("downloads/task-1") {
		t.Fatalf("OutputRoot = %q", view.OutputRoot)
	}
	if view.ThumbnailRoot != filepath.Clean("thumbnails/task-1") {
		t.Fatalf("ThumbnailRoot = %q", view.ThumbnailRoot)
	}
	if view.ThumbnailPath != filepath.Clean("thumbnails/task-1/thumb.jpg") {
		t.Fatalf("ThumbnailPath = %q", view.ThumbnailPath)
	}
	if view.ReportPath != filepath.Clean("report.json") {
		t.Fatalf("ReportPath = %q", view.ReportPath)
	}
}

func TestLoadTaskViewsSortsNewestFirst(t *testing.T) {
	workspace := t.TempDir()
	paths := runtime.NewPaths(workspace)
	if err := paths.Ensure(); err != nil {
		t.Fatalf("paths.Ensure() error = %v", err)
	}
	writeReport := func(taskID string, createdAt time.Time) {
		reportPath := paths.TaskReportPath(taskID)
		if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
			t.Fatalf("mkdir report dir: %v", err)
		}
		report := tasks.TaskReport{
			TaskID: taskID,
			Manifest: siteflow.TaskManifestSummary{
				Site:       "browser",
				Title:      "task " + taskID,
				PrimaryURL: "https://example.com/" + taskID,
				AssetCount: 1,
			},
			State:     tasks.TaskStatePrepared,
			CreatedAt: createdAt,
			StartedAt: createdAt,
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatalf("marshal report: %v", err)
		}
		if err := os.WriteFile(reportPath, data, 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}
	}

	now := time.Now().UTC()
	writeReport("1", now.Add(-time.Hour))
	writeReport("2", now)

	views, err := LoadTaskViews(paths)
	if err != nil {
		t.Fatalf("LoadTaskViews() error = %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("len(views) = %d", len(views))
	}
	if views[0].TaskID != "2" {
		t.Fatalf("first view task id = %q, want 2", views[0].TaskID)
	}
	if views[1].TaskID != "1" {
		t.Fatalf("second view task id = %q, want 1", views[1].TaskID)
	}
}
