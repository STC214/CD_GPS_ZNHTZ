package ui

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/tasks"
)

// LoadLegacyComicDownloaderHistory reads the old runtime/comic_downloader_state.json file
// and merges the current todo list with a read-only snapshot of those tasks.
func (l *TodoList) LoadLegacyComicDownloaderHistory(path string) (int, error) {
	state, err := projectruntime.LoadLegacyComicDownloaderState(path)
	if err != nil {
		if strings.TrimSpace(path) != "" && isNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return l.ImportLegacyComicDownloaderState(state), nil
}

// PreviewLegacyComicDownloaderState reports how many legacy tasks would be added
// and how many would be skipped as duplicates if the given snapshot were imported.
func (l *TodoList) PreviewLegacyComicDownloaderState(state projectruntime.LegacyComicDownloaderState) (added int, duplicates int) {
	l.mu.Lock()
	currentKeys := make(map[string]struct{}, len(l.items))
	for _, item := range l.items {
		currentKeys[legacyHistoryImportKey(item.Request)] = struct{}{}
	}
	l.mu.Unlock()

	for _, legacy := range state.Tasks {
		item := legacyTaskToTodoItem(legacy)
		key := legacyHistoryImportKey(item.Request)
		if _, exists := currentKeys[key]; exists {
			duplicates++
			continue
		}
		currentKeys[key] = struct{}{}
		added++
	}
	return added, duplicates
}

// ImportLegacyComicDownloaderState merges the current todo list with a snapshot from the old app state file.
func (l *TodoList) ImportLegacyComicDownloaderState(state projectruntime.LegacyComicDownloaderState) int {
	l.mu.Lock()
	existingKeys := make(map[string]struct{}, len(l.items))
	maxSeq := l.seq
	for _, item := range l.items {
		existingKeys[legacyHistoryImportKey(item.Request)] = struct{}{}
		if itemSeq := todoItemSequence(item.ID); itemSeq > maxSeq {
			maxSeq = itemSeq
		}
	}
	l.mu.Unlock()

	items := make([]TodoItem, 0, len(state.Tasks))
	paths := projectruntime.NewPaths(strings.TrimSpace(l.runtimeRoot))
	for _, legacy := range state.Tasks {
		item := legacyTaskToTodoItem(legacy)
		item.Request.DownloadRoot = projectruntime.ResolvePath(paths.Root, item.Request.DownloadRoot)
		item.Request.OutputDir = projectruntime.ResolvePath(paths.Root, item.Request.OutputDir)
		item.Result.DownloadedDir = projectruntime.ResolvePath(paths.Root, item.Result.DownloadedDir)
		item.Result.ThumbnailPath = projectruntime.ResolvePath(paths.Root, item.Result.ThumbnailPath)
		if resolved := resolveImportedLegacyThumbnailPath(paths, item.ID, item.Result.ThumbnailPath); resolved != "" {
			item.Result.ThumbnailPath = resolved
		}
		key := legacyHistoryImportKey(item.Request)
		if _, exists := existingKeys[key]; exists {
			continue
		}
		existingKeys[key] = struct{}{}
		items = append(items, item)
		if itemSeq := todoItemSequence(item.ID); itemSeq > maxSeq {
			maxSeq = itemSeq
		}
	}

	seq := maxSeq
	if state.NextTaskID > 0 && state.NextTaskID-1 > seq {
		seq = state.NextTaskID - 1
	}

	l.mu.Lock()
	l.items = append(l.items, items...)
	l.seq = seq
	runtimeRoot := strings.TrimSpace(l.runtimeRoot)
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	if runtimeRoot != "" {
		for _, item := range items {
			if err := SaveTaskReport(runtimeRoot, item); err != nil {
				log.Printf("save imported legacy task report failed: %v", err)
			}
		}
	}
	return len(items)
}

func resolveImportedLegacyThumbnailPath(paths projectruntime.Paths, taskID, storedPath string) string {
	storedPath = strings.TrimSpace(storedPath)
	if storedPath != "" {
		if resolved := projectruntime.ResolvePath(paths.Root, storedPath); resolved != "" {
			if _, err := os.Stat(resolved); err == nil {
				return resolved
			}
		}
	}
	if taskID == "" {
		return storedPath
	}
	candidate := paths.TaskThumbnailPath(taskID)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	if storedPath != "" {
		return projectruntime.ResolvePath(paths.Root, storedPath)
	}
	return candidate
}

// ExportLegacyComicDownloaderState converts the current todo list into the legacy snapshot format.
func (l *TodoList) ExportLegacyComicDownloaderState(concurrency int) projectruntime.LegacyComicDownloaderState {
	l.mu.Lock()
	items := make([]TodoItem, len(l.items))
	copy(items, l.items)
	l.mu.Unlock()
	tasksOut := make([]projectruntime.LegacyComicDownloaderTask, 0, len(items))
	maxID := 0
	runtimeRoot := strings.TrimSpace(l.runtimeRoot)
	for idx := len(items) - 1; idx >= 0; idx-- {
		item := items[idx]
		legacyTask, id := todoItemToLegacyTask(item, len(items)-idx, runtimeRoot)
		if id > maxID {
			maxID = id
		}
		tasksOut = append(tasksOut, legacyTask)
	}
	if maxID < len(tasksOut) {
		maxID = len(tasksOut)
	}
	return projectruntime.LegacyComicDownloaderState{
		Version:     1,
		NextTaskID:  maxID + 1,
		Concurrency: concurrency,
		Tasks:       tasksOut,
	}
}

// SaveLegacyComicDownloaderState writes the current todo list into the legacy snapshot file.
func (l *TodoList) SaveLegacyComicDownloaderState(path string, concurrency int) error {
	state := l.ExportLegacyComicDownloaderState(concurrency)
	return projectruntime.SaveLegacyComicDownloaderState(path, state)
}

func legacyHistoryImportKey(req tasks.BrowserLaunchRequest) string {
	req = req.Normalize()
	if url := strings.TrimSpace(req.URL); url != "" {
		return strings.ToLower(url)
	}
	return strings.ToLower(strings.TrimSpace(req.BrowserType))
}

func legacyTaskToTodoItem(task projectruntime.LegacyComicDownloaderTask) TodoItem {
	req := tasks.BrowserLaunchRequest{
		URL:          tasks.NormalizeTaskURL(task.URL),
		BrowserType:  legacyBrowserType(task),
		Headless:     task.Headless,
		RuntimeRoot:  "runtime",
		DownloadRoot: strings.TrimSpace(task.DownloadRoot),
		OutputDir:    strings.TrimSpace(task.OutputDir),
	}
	req = req.Normalize()
	status := legacyStateToTodoStatus(task.State)
	return TodoItem{
		ID:          fmt.Sprintf("legacy-%d", task.ID),
		Request:     req,
		Status:      status,
		Progress:    legacyProgress(task),
		Phase:       legacyPhase(task.State),
		StepCurrent: legacyStepCurrent(task),
		StepTotal:   legacyStepTotal(task),
		StepMessage: legacyStepMessage(task),
		CreatedAt:   legacyTimeOrNow(task.CreatedAt, task.UpdatedAt),
		StartedAt:   legacyStartedAt(task),
		FinishedAt:  legacyFinishedAt(task),
		Result: tasks.BrowserRunResult{
			URL:                tasks.NormalizeTaskURL(task.URL),
			Title:              tasks.NormalizeTaskURL(task.Title),
			BrowserType:        req.BrowserType,
			Headless:           task.Headless,
			Site:               legacySite(task),
			ResolvedURL:        tasks.NormalizeTaskURL(task.URL),
			PageType:           legacyPageType(task),
			Blocked:            strings.Contains(strings.ToLower(task.Detail), "blocked"),
			VerificationNeeded: strings.Contains(strings.ToLower(task.Detail), "verification"),
			MatchedMarker:      "",
			Note:               task.Detail,
			DownloadedDir:      strings.TrimSpace(task.OutputDir),
			ThumbnailPath:      strings.TrimSpace(task.ThumbnailPath),
		},
		LastError: legacyLastError(task),
	}
}

func todoItemToLegacyTask(item TodoItem, fallbackID int, runtimeRoot string) (projectruntime.LegacyComicDownloaderTask, int) {
	id := todoItemSequence(item.ID)
	if id <= 0 {
		id = fallbackID
	}
	title := strings.TrimSpace(item.Result.Title)
	if title == "" {
		title = strings.TrimSpace(item.Request.URL)
	}
	if title == "" {
		title = item.ID
	}
	outputDir := strings.TrimSpace(item.Result.DownloadedDir)
	if outputDir == "" {
		outputDir = strings.TrimSpace(item.Request.OutputDir)
	}
	detail := strings.TrimSpace(item.StepMessage)
	if detail == "" {
		detail = strings.TrimSpace(item.Result.Note)
	}
	if detail == "" {
		detail = strings.TrimSpace(item.LastError)
	}
	updatedAt := item.FinishedAt
	if updatedAt.IsZero() {
		updatedAt = item.StartedAt
	}
	if updatedAt.IsZero() {
		updatedAt = item.CreatedAt
	}
	worker := strings.TrimSpace(item.Request.WorkerID)
	if worker == "" {
		worker = "ui"
	}
	workerSource := strings.TrimSpace(item.Request.BrowserType)
	if workerSource == "" {
		workerSource = legacyBrowserTypeForStatus(item.Status)
	}
	return projectruntime.LegacyComicDownloaderTask{
		ID:            id,
		URL:           strings.TrimSpace(item.Request.URL),
		Title:         title,
		DownloadRoot:  projectruntime.RelativizePath(runtimeRoot, strings.TrimSpace(item.Request.DownloadRoot)),
		OutputDir:     projectruntime.RelativizePath(runtimeRoot, outputDir),
		Headless:      item.Request.Headless,
		HTTPOnly:      false,
		ThumbnailPath: projectruntime.RelativizePath(runtimeRoot, strings.TrimSpace(item.Result.ThumbnailPath)),
		Worker:        worker,
		WorkerSource:  workerSource,
		State:         string(item.Status),
		Detail:        detail,
		Percent:       item.Progress,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     updatedAt,
	}, id
}

func legacyBrowserTypeForStatus(status TodoStatus) string {
	switch status {
	case TodoStatusPending, TodoStatusQueued, TodoStatusRouting, TodoStatusPreparing, TodoStatusRunning, TodoStatusPaused, TodoStatusWaitingVerification, TodoStatusVerificationCleared, TodoStatusCompleted, TodoStatusFailed:
		return string(projectruntime.BrowserTypeFirefox)
	default:
		return string(projectruntime.BrowserTypeFirefox)
	}
}

func legacyBrowserType(task projectruntime.LegacyComicDownloaderTask) string {
	worker := strings.ToLower(strings.TrimSpace(task.Worker + " " + task.WorkerSource))
	switch {
	case strings.Contains(worker, "myreading"):
		return string(projectruntime.BrowserTypeFirefox)
	case strings.Contains(worker, "nyahentai"):
		return string(projectruntime.BrowserTypeFirefox)
	case strings.Contains(worker, "zeri"):
		return string(projectruntime.BrowserTypeFirefox)
	default:
		return string(projectruntime.BrowserTypeFirefox)
	}
}

func legacyStateToTodoStatus(state string) TodoStatus {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "queued":
		return TodoStatusQueued
	case "routing":
		return TodoStatusRouting
	case "preparing":
		return TodoStatusPreparing
	case "prepared", "running":
		return TodoStatusRunning
	case "paused":
		return TodoStatusPaused
	case "waiting_verification":
		return TodoStatusWaitingVerification
	case "verification_cleared":
		return TodoStatusVerificationCleared
	case "completed", "done":
		return TodoStatusCompleted
	case "failed", "error":
		return TodoStatusFailed
	default:
		return TodoStatusPending
	}
}

func legacyPhase(state string) string {
	switch legacyStateToTodoStatus(state) {
	case TodoStatusQueued:
		return "queued"
	case TodoStatusRouting:
		return "routing"
	case TodoStatusPreparing:
		return "preparing"
	case TodoStatusRunning:
		return "running"
	case TodoStatusPaused:
		return "paused"
	case TodoStatusWaitingVerification:
		return "waiting_verification"
	case TodoStatusVerificationCleared:
		return "verification_cleared"
	case TodoStatusCompleted:
		return "completed"
	case TodoStatusFailed:
		return "failed"
	default:
		return "pending"
	}
}

func legacyProgress(task projectruntime.LegacyComicDownloaderTask) float64 {
	if task.Percent > 1 {
		return 1
	}
	if task.Percent < 0 {
		return 0
	}
	return task.Percent
}

func legacyStepCurrent(task projectruntime.LegacyComicDownloaderTask) int {
	if p := legacyProgress(task); p >= 1 {
		return 1
	}
	if legacyProgress(task) > 0 {
		return 1
	}
	return 0
}

func legacyStepTotal(task projectruntime.LegacyComicDownloaderTask) int {
	if legacyProgress(task) > 0 {
		return 1
	}
	return 0
}

func legacyStepMessage(task projectruntime.LegacyComicDownloaderTask) string {
	if strings.TrimSpace(task.Detail) != "" {
		return task.Detail
	}
	return legacyPhase(task.State)
}

func legacyLastError(task projectruntime.LegacyComicDownloaderTask) string {
	if legacyStateToTodoStatus(task.State) != TodoStatusFailed {
		return ""
	}
	if strings.TrimSpace(task.Detail) != "" {
		return task.Detail
	}
	return "failed"
}

func legacySite(task projectruntime.LegacyComicDownloaderTask) string {
	worker := strings.ToLower(strings.TrimSpace(task.Worker + " " + task.WorkerSource))
	switch {
	case strings.Contains(worker, "myreading"):
		return "myreadingmanga"
	case strings.Contains(worker, "nyahentai"):
		return "nyahentai"
	case strings.Contains(worker, "zeri"):
		return "zeri"
	default:
		return strings.TrimSpace(task.Worker)
	}
}

func legacyPageType(task projectruntime.LegacyComicDownloaderTask) string {
	switch legacyStateToTodoStatus(task.State) {
	case TodoStatusCompleted, TodoStatusFailed:
		return "content"
	default:
		return "summary"
	}
}

func legacyTimeOrNow(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary.UTC()
	}
	if !fallback.IsZero() {
		return fallback.UTC()
	}
	return time.Now().UTC()
}

func legacyStartedAt(task projectruntime.LegacyComicDownloaderTask) time.Time {
	switch legacyStateToTodoStatus(task.State) {
	case TodoStatusQueued, TodoStatusPending:
		return time.Time{}
	default:
		return legacyTimeOrNow(task.CreatedAt, task.UpdatedAt)
	}
}

func legacyFinishedAt(task projectruntime.LegacyComicDownloaderTask) time.Time {
	switch legacyStateToTodoStatus(task.State) {
	case TodoStatusCompleted, TodoStatusFailed:
		if !task.UpdatedAt.IsZero() {
			return task.UpdatedAt.UTC()
		}
		return legacyTimeOrNow(task.CreatedAt, task.UpdatedAt)
	default:
		return time.Time{}
	}
}

func todoItemSequence(id string) int {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0
	}
	if strings.HasPrefix(id, "legacy-") {
		id = strings.TrimPrefix(id, "legacy-")
	}
	if strings.HasPrefix(id, "todo-") {
		id = strings.TrimPrefix(id, "todo-")
	}
	n, err := strconv.Atoi(id)
	if err != nil {
		return 0
	}
	return n
}

func isNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
