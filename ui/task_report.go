package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/siteflow"
	"comic_downloader_go_playwright_stealth/tasks"
)

// SaveTaskReport writes the current task snapshot to the runtime task report path.
func SaveTaskReport(runtimeRoot string, item TodoItem) error {
	runtimeRoot = strings.TrimSpace(runtimeRoot)
	if runtimeRoot == "" {
		runtimeRoot = "runtime"
	}
	paths := projectruntime.NewPathsFromRuntimeRoot(runtimeRoot)
	reportPath := paths.TaskReportPath(item.ID)
	report := taskReportFromItem(paths, item, reportPath)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task report %q: %w", reportPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return fmt.Errorf("create task report dir %q: %w", filepath.Dir(reportPath), err)
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return fmt.Errorf("write task report %q: %w", reportPath, err)
	}
	if err := writeTaskLog(report, data, paths.TaskLogPath(item.ID)); err != nil {
		return err
	}
	return nil
}

// CleanupTaskLog removes the human-readable task log for one task.
func CleanupTaskLog(runtimeRoot, taskID string) error {
	runtimeRoot = strings.TrimSpace(runtimeRoot)
	if runtimeRoot == "" {
		runtimeRoot = "runtime"
	}
	paths := projectruntime.NewPathsFromRuntimeRoot(runtimeRoot)
	logPath := paths.TaskLogPath(taskID)
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove task log %q: %w", logPath, err)
	}
	return nil
}

func writeTaskLog(report tasks.TaskReport, reportJSON []byte, logPath string) error {
	logPath = strings.TrimSpace(logPath)
	if logPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create task log dir %q: %w", filepath.Dir(logPath), err)
	}
	var b strings.Builder
	b.WriteString("Task Report\n")
	b.WriteString("Generated: ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339Nano))
	b.WriteString("\n")
	b.WriteString("TaskID: ")
	b.WriteString(report.TaskID)
	b.WriteString("\n")
	b.WriteString("State: ")
	b.WriteString(string(report.State))
	b.WriteString("\n")
	b.WriteString("Title: ")
	b.WriteString(report.Manifest.Title)
	b.WriteString("\n")
	b.WriteString("Site: ")
	b.WriteString(report.Manifest.Site)
	b.WriteString("\n")
	b.WriteString("PrimaryURL: ")
	b.WriteString(report.Manifest.PrimaryURL)
	b.WriteString("\n")
	b.WriteString("OutputRoot: ")
	b.WriteString(report.OutputRoot)
	b.WriteString("\n")
	b.WriteString("ThumbnailPath: ")
	b.WriteString(report.ThumbnailPath)
	b.WriteString("\n")
	b.WriteString("LastError: ")
	b.WriteString(report.LastError)
	b.WriteString("\n\nJSON Snapshot:\n")
	b.Write(reportJSON)
	b.WriteString("\n")
	if err := os.WriteFile(logPath, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write task log %q: %w", logPath, err)
	}
	return nil
}

// TaskDetailsFromItem builds a detail view directly from the current item snapshot.
func TaskDetailsFromItem(item TodoItem, reportPath string) TaskDetails {
	reportRoot := runtimeRootFromReportPath(reportPath)
	title := strings.TrimSpace(item.Result.Title)
	if title == "" {
		title = strings.TrimSpace(item.Request.URL)
	}
	if title == "" {
		title = item.ID
	}
	primaryURL := strings.TrimSpace(item.Result.URL)
	if primaryURL == "" {
		primaryURL = strings.TrimSpace(item.Request.URL)
	}
	downloadRoot := strings.TrimSpace(item.Result.DownloadedDir)
	if downloadRoot == "" {
		downloadRoot = strings.TrimSpace(item.Request.OutputDir)
	}
	downloadRoot = projectruntime.ResolvePath(reportRoot, downloadRoot)
	thumbnailPath := resolveTaskThumbnailPath(reportPath, item.ID, item.Result.ThumbnailPath)
	reportPath = projectruntime.ResolvePath(reportRoot, reportPath)
	view := TaskView{
		TaskID:             item.ID,
		Title:              title,
		Site:               strings.TrimSpace(item.Result.Site),
		State:              string(item.Status),
		Verification:       strings.TrimSpace(item.Result.Note),
		PrimaryURL:         primaryURL,
		BrowserType:        strings.TrimSpace(item.Result.BrowserType),
		BrowserPath:        strings.TrimSpace(item.Result.BrowserPath),
		BrowserMode:        strings.TrimSpace(item.Result.BrowserMode),
		PageType:           strings.TrimSpace(item.Result.PageType),
		OutputRoot:         downloadRoot,
		ThumbnailPath:      thumbnailPath,
		ReportPath:         reportPath,
		CreatedAt:          item.CreatedAt,
		StartedAt:          item.StartedAt,
		FinishedAt:         item.FinishedAt,
		LastError:          strings.TrimSpace(item.LastError),
		Blocked:            item.Result.Blocked,
		Verified:           item.Result.Verified,
		VerificationNeeded: item.Result.VerificationNeeded,
		MatchedMarker:      item.Result.MatchedMarker,
		Note:               strings.TrimSpace(item.Result.Note),
		AssetCount:         item.Result.DownloadedCount,
	}
	return TaskDetails{
		TaskView:          view,
		DownloadRoot:      downloadRoot,
		StorageState:      "",
		VerificationState: "",
		InitScript:        "",
		ExtraLogPaths:     nil,
	}
}

func taskReportFromItem(paths projectruntime.Paths, item TodoItem, reportPath string) tasks.TaskReport {
	reportRoot := paths.Root
	title := strings.TrimSpace(item.Result.Title)
	if title == "" {
		title = strings.TrimSpace(item.Request.URL)
	}
	if title == "" {
		title = item.ID
	}
	manifest := siteflow.TaskManifestSummary{
		Site:       strings.TrimSpace(item.Result.Site),
		Title:      title,
		PrimaryURL: strings.TrimSpace(item.Request.URL),
		AssetCount: item.Result.DownloadedCount,
		Blocked:    item.Result.Blocked,
	}
	downloadRoot := strings.TrimSpace(item.Result.DownloadedDir)
	if downloadRoot == "" {
		downloadRoot = strings.TrimSpace(item.Request.OutputDir)
	}
	thumbnailRoot := strings.TrimSpace(filepath.Dir(item.Result.ThumbnailPath))
	if thumbnailRoot == "" || thumbnailRoot == "." {
		thumbnailRoot = paths.ThumbnailsRoot
	}
	statePath := filepath.Join(paths.Root, "comic_downloader_state.json")
	logPath := paths.TaskLogPath(item.ID)
	return tasks.TaskReport{
		TaskID:             item.ID,
		Manifest:           manifest,
		State:              taskStateForItem(item.Status),
		Verification:       string(item.Status),
		BrowserType:        strings.TrimSpace(item.Result.BrowserType),
		BrowserPath:        strings.TrimSpace(item.Result.BrowserPath),
		BrowserMode:        strings.TrimSpace(item.Result.BrowserMode),
		PageType:           strings.TrimSpace(item.Result.PageType),
		Verified:           item.Result.Verified,
		VerificationNeeded: item.Result.VerificationNeeded,
		Blocked:            item.Result.Blocked,
		MatchedMarker:      strings.TrimSpace(item.Result.MatchedMarker),
		Note:               strings.TrimSpace(item.Result.Note),
		OutputRoot:         projectruntime.RelativizePath(reportRoot, downloadRoot),
		ThumbnailRoot:      projectruntime.RelativizePath(reportRoot, thumbnailRoot),
		ThumbnailPath:      projectruntime.RelativizePath(reportRoot, strings.TrimSpace(item.Result.ThumbnailPath)),
		StatePath:          projectruntime.RelativizePath(reportRoot, statePath),
		ReportPath:         projectruntime.RelativizePath(reportRoot, reportPath),
		LogPath:            projectruntime.RelativizePath(reportRoot, logPath),
		StorageState:       "",
		VerificationState:  "",
		InitScript:         "",
		CreatedAt:          item.CreatedAt,
		StartedAt:          item.StartedAt,
		FinishedAt:         item.FinishedAt,
		LastError:          strings.TrimSpace(item.LastError),
	}
}

func taskStateForItem(status TodoStatus) tasks.TaskState {
	switch status {
	case TodoStatusCompleted:
		return tasks.TaskStateCompleted
	case TodoStatusFailed:
		return tasks.TaskStateFailed
	case TodoStatusPaused:
		return tasks.TaskStatePaused
	case TodoStatusWaitingVerification:
		return tasks.TaskStateWaitingVerification
	case TodoStatusVerificationCleared:
		return tasks.TaskStateVerificationCleared
	case TodoStatusRunning:
		return tasks.TaskStateRunning
	case TodoStatusRouting:
		return tasks.TaskStateRouting
	case TodoStatusPreparing:
		return tasks.TaskStatePreparing
	case TodoStatusQueued:
		return tasks.TaskStateQueued
	default:
		return tasks.TaskStatePrepared
	}
}

func sanitizeTaskReportPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "task"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			fallthrough
		case r >= 'A' && r <= 'Z':
			fallthrough
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
