package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/tasks"
)

// TaskView is the UI-ready surface for one task entry.
type TaskView struct {
	TaskID             string
	Title              string
	Site               string
	State              string
	Verification       string
	PrimaryURL         string
	BrowserType        string
	BrowserPath        string
	BrowserMode        string
	PageType           string
	OutputRoot         string
	ThumbnailRoot      string
	ThumbnailPath      string
	StatePath          string
	ReportPath         string
	LogPath            string
	CreatedAt          time.Time
	StartedAt          time.Time
	FinishedAt         time.Time
	LastError          string
	Blocked            bool
	Verified           bool
	VerificationNeeded bool
	MatchedMarker      string
	Note               string
	AssetCount         int
}

// TaskDetails is the full UI detail surface for a task.
type TaskDetails struct {
	TaskView
	DownloadRoot      string
	StorageState      string
	VerificationState string
	InitScript        string
	ExtraLogPaths     []string
}

// LoadTaskView loads a single task report file into a UI-ready model.
func LoadTaskView(reportPath string) (TaskView, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return TaskView{}, fmt.Errorf("read task report %q: %w", reportPath, err)
	}
	var report tasks.TaskReport
	if err := json.Unmarshal(data, &report); err != nil {
		return TaskView{}, fmt.Errorf("unmarshal task report %q: %w", reportPath, err)
	}
	reportRoot := runtimeRootFromReportPath(reportPath)
	thumbnailPath := resolveStoredTaskPath(reportRoot, reportPath, report.ThumbnailPath)
	return TaskView{
		TaskID:             report.TaskID,
		Title:              report.Manifest.Title,
		Site:               report.Manifest.Site,
		State:              string(report.State),
		Verification:       string(report.Verification),
		PrimaryURL:         report.Manifest.PrimaryURL,
		BrowserType:        report.BrowserType,
		BrowserPath:        report.BrowserPath,
		BrowserMode:        report.BrowserMode,
		PageType:           report.PageType,
		OutputRoot:         resolveStoredTaskPath(reportRoot, reportPath, report.OutputRoot),
		ThumbnailRoot:      resolveStoredTaskPath(reportRoot, reportPath, report.ThumbnailRoot),
		ThumbnailPath:      thumbnailPath,
		StatePath:          resolveStoredTaskPath(reportRoot, reportPath, report.StatePath),
		ReportPath:         resolveStoredTaskPath(reportRoot, reportPath, report.ReportPath),
		LogPath:            resolveStoredTaskPath(reportRoot, reportPath, report.LogPath),
		CreatedAt:          report.CreatedAt,
		StartedAt:          report.StartedAt,
		FinishedAt:         report.FinishedAt,
		LastError:          report.LastError,
		Blocked:            report.Manifest.Blocked,
		Verified:           report.Verified,
		VerificationNeeded: report.VerificationNeeded,
		MatchedMarker:      report.MatchedMarker,
		Note:               report.Note,
		AssetCount:         report.Manifest.AssetCount,
	}, nil
}

func resolveStoredTaskPath(reportRoot, reportPath, storedPath string) string {
	storedPath = strings.TrimSpace(storedPath)
	if storedPath == "" {
		return ""
	}
	if filepath.IsAbs(storedPath) {
		return filepath.Clean(storedPath)
	}
	base := strings.TrimSpace(reportRoot)
	if base == "" {
		base = runtimeRootFromReportPath(reportPath)
	}
	if base != "" {
		return filepath.Clean(filepath.Join(base, storedPath))
	}
	return filepath.Clean(storedPath)
}

func resolveTaskThumbnailPath(reportPath, taskID, storedPath string) string {
	storedPath = strings.TrimSpace(storedPath)
	taskID = strings.TrimSpace(taskID)
	reportPath = strings.TrimSpace(reportPath)
	if storedPath != "" {
		if resolved := resolveStoredTaskPath("", reportPath, storedPath); resolved != "" {
			if _, err := os.Stat(resolved); err == nil {
				return resolved
			}
		}
	}
	if taskID != "" {
		if runtimeRoot := runtimeRootFromReportPath(reportPath); runtimeRoot != "" {
			roots := []string{runtimeRoot}
			if filepath.Base(runtimeRoot) == "runtime" {
				roots = append(roots, filepath.Dir(runtimeRoot))
			}
			for _, root := range roots {
				paths := runtime.NewPaths(root)
				candidate := paths.TaskThumbnailPath(taskID)
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
	}
	if storedPath != "" {
		return resolveStoredTaskPath("", reportPath, storedPath)
	}
	return ""
}

func runtimeRootFromReportPath(reportPath string) string {
	reportPath = strings.TrimSpace(reportPath)
	if reportPath == "" {
		return ""
	}
	reportDir := filepath.Clean(filepath.Dir(reportPath))
	taskDir := filepath.Base(reportDir)
	tasksDir := filepath.Base(filepath.Dir(reportDir))
	if !strings.HasPrefix(taskDir, "task-") || tasksDir != "tasks" {
		return ""
	}
	return filepath.Clean(filepath.Dir(filepath.Dir(reportDir)))
}

// LoadTaskDetails loads the full detail view for a task report.
func LoadTaskDetails(reportPath string) (TaskDetails, error) {
	view, err := LoadTaskView(reportPath)
	if err != nil {
		return TaskDetails{}, err
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return TaskDetails{}, fmt.Errorf("read task report %q: %w", reportPath, err)
	}
	var report tasks.TaskReport
	if err := json.Unmarshal(data, &report); err != nil {
		return TaskDetails{}, fmt.Errorf("unmarshal task report %q: %w", reportPath, err)
	}
	return TaskDetails{
		TaskView:          view,
		DownloadRoot:      report.OutputRoot,
		StorageState:      report.StorageState,
		VerificationState: report.VerificationState,
		InitScript:        report.InitScript,
		ExtraLogPaths:     taskExtraLogPaths(view),
	}, nil
}

func taskExtraLogPaths(view TaskView) []string {
	extra := make([]string, 0, 1)
	if view.Site == "nyahentai" && view.OutputRoot != "" {
		extra = append(extra, filepath.Join(view.OutputRoot, "nyahentai-trace.log"))
	}
	return extra
}

// LoadTaskViews scans the runtime task directory and returns all task reports.
func LoadTaskViews(paths runtime.Paths) ([]TaskView, error) {
	pattern := filepath.Join(paths.TasksRoot, "task-*", "report.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob task reports %q: %w", pattern, err)
	}
	views := make([]TaskView, 0, len(matches))
	for _, reportPath := range matches {
		view, err := LoadTaskView(reportPath)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	sort.SliceStable(views, func(i, j int) bool {
		if views[i].CreatedAt.Equal(views[j].CreatedAt) {
			return views[i].TaskID < views[j].TaskID
		}
		return views[i].CreatedAt.After(views[j].CreatedAt)
	})
	return views, nil
}
