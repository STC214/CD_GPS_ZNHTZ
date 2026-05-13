package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/siteflow/zeri"
	"comic_downloader_go_playwright_stealth/tasks"
)

// TodoStatus is the frontend queue state for an item before it is started.
type TodoStatus string

const (
	TodoStatusPending             TodoStatus = "pending"
	TodoStatusQueued              TodoStatus = "queued"
	TodoStatusRouting             TodoStatus = "routing"
	TodoStatusPreparing           TodoStatus = "preparing"
	TodoStatusRunning             TodoStatus = "running"
	TodoStatusPaused              TodoStatus = "paused"
	TodoStatusWaitingVerification TodoStatus = "waiting_verification"
	TodoStatusVerificationCleared TodoStatus = "verification_cleared"
	TodoStatusCompleted           TodoStatus = "completed"
	TodoStatusFailed              TodoStatus = "failed"
)

// TodoItem is one item in the frontend todo list.
type TodoItem struct {
	ID          string                     `json:"id"`
	Request     tasks.BrowserLaunchRequest `json:"request"`
	Status      TodoStatus                 `json:"status"`
	Progress    float64                    `json:"progress,omitempty"`
	Phase       string                     `json:"phase,omitempty"`
	StepCurrent int                        `json:"stepCurrent,omitempty"`
	StepTotal   int                        `json:"stepTotal,omitempty"`
	StepMessage string                     `json:"stepMessage,omitempty"`
	StepBytes   int64                      `json:"stepBytes,omitempty"`
	StepBPS     float64                    `json:"stepBps,omitempty"`
	AverageBPS  float64                    `json:"averageBps,omitempty"`
	CreatedAt   time.Time                  `json:"createdAt"`
	StartedAt   time.Time                  `json:"startedAt"`
	FinishedAt  time.Time                  `json:"finishedAt"`
	Result      tasks.BrowserRunResult     `json:"result,omitempty"`
	LastError   string                     `json:"lastError,omitempty"`
}

// TodoList manages items that are added as pending and started later in batch.
type TodoList struct {
	mu                sync.Mutex
	cond              *sync.Cond
	seq               int
	items             []TodoItem
	notifier          func()
	runtimeRoot       string
	maxParallel       int
	running           int
	progressNotifyMu  sync.Mutex
	progressNotifyAt  time.Time
	progressNotifyRun bool
	progressDelay     time.Duration
	activeCancels     map[string]context.CancelFunc
}

// NewTodoList builds an empty todo list.
func NewTodoList() *TodoList {
	l := &TodoList{maxParallel: 1, progressDelay: 80 * time.Millisecond, activeCancels: map[string]context.CancelFunc{}}
	l.cond = sync.NewCond(&l.mu)
	return l
}

// SetRuntimeRoot sets the persistent runtime root used for task artifacts.
func (l *TodoList) SetRuntimeRoot(runtimeRoot string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.runtimeRoot = strings.TrimSpace(runtimeRoot)
}

// SetConcurrencyLimit updates the maximum number of tasks allowed to run at once.
func (l *TodoList) SetConcurrencyLimit(maxParallel int) {
	if maxParallel <= 0 {
		maxParallel = 1
	}
	l.mu.Lock()
	if l.cond == nil {
		l.cond = sync.NewCond(&l.mu)
	}
	l.maxParallel = maxParallel
	l.cond.Broadcast()
	l.mu.Unlock()
}

// SetProgressNotifyDelay updates the minimum delay between task list refresh notifications.
func (l *TodoList) SetProgressNotifyDelay(delay time.Duration) {
	if delay <= 0 {
		delay = 80 * time.Millisecond
	}
	l.progressNotifyMu.Lock()
	l.progressDelay = delay
	l.progressNotifyMu.Unlock()
}

// SetNotifier sets a callback that is invoked whenever the list changes.
func (l *TodoList) SetNotifier(fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.notifier = fn
}

// AddPending appends one browser request in pending state.
func (l *TodoList) AddPending(req tasks.BrowserLaunchRequest) TodoItem {
	req = req.Normalize()
	l.mu.Lock()
	l.seq++
	item := TodoItem{
		ID:        fmt.Sprintf("todo-%03d", l.seq),
		Request:   req,
		Status:    TodoStatusPending,
		Progress:  0,
		Phase:     "pending",
		CreatedAt: time.Now().UTC(),
	}
	l.items = append([]TodoItem{item}, l.items...)
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	return item
}

// FindDuplicate returns the first existing item that matches the request's identity.
func (l *TodoList) FindDuplicate(req tasks.BrowserLaunchRequest) (TodoItem, bool) {
	req = req.Normalize()
	key := browserLaunchRequestKey(req)
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, item := range l.items {
		if browserLaunchRequestKey(item.Request) == key {
			return item, true
		}
	}
	return TodoItem{}, false
}

// Items returns a copy of the current todo list.
func (l *TodoList) Items() []TodoItem {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sortedItemsLocked()
}

// Count returns the current number of items.
func (l *TodoList) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.items)
}

// ItemsRange returns a copy of the requested inclusive-exclusive item range.
func (l *TodoList) ItemsRange(start, end int) []TodoItem {
	l.mu.Lock()
	defer l.mu.Unlock()
	sorted := l.sortedItemsLocked()
	if start < 0 {
		start = 0
	}
	if end > len(sorted) {
		end = len(sorted)
	}
	if start >= end {
		return nil
	}
	window := make([]TodoItem, end-start)
	copy(window, sorted[start:end])
	return window
}

// Pending returns all items that have not been started yet.
func (l *TodoList) Pending() []TodoItem {
	l.mu.Lock()
	defer l.mu.Unlock()
	pending := make([]TodoItem, 0, len(l.items))
	for _, item := range l.sortedItemsLocked() {
		if item.Status == TodoStatusPending {
			pending = append(pending, item)
		}
	}
	return pending
}

// ClearFinished removes completed and failed items from the list.
func (l *TodoList) ClearFinished() int {
	l.mu.Lock()
	kept := l.items[:0]
	removed := 0
	for _, item := range l.items {
		switch item.Status {
		case TodoStatusCompleted, TodoStatusFailed:
			removed++
		default:
			kept = append(kept, item)
		}
	}
	l.items = kept
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	return removed
}

// ItemByID returns one item by identifier.
func (l *TodoList) ItemByID(id string) (TodoItem, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	index, ok := l.findItemIndexByIDLocked(id)
	if !ok {
		return TodoItem{}, false
	}
	return l.items[index], true
}

// RemoveByIDs removes all matching items from the list.
func (l *TodoList) RemoveByIDs(ids []string) int {
	if len(ids) == 0 {
		return 0
	}
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return 0
	}
	l.mu.Lock()
	kept := l.items[:0]
	removed := 0
	for _, item := range l.items {
		if _, ok := idSet[item.ID]; ok {
			if todoStatusIsActive(item.Status) {
				kept = append(kept, item)
				continue
			}
			removed++
			continue
		}
		kept = append(kept, item)
	}
	l.items = kept
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	return removed
}

// SetStatusByIDs sets the status and phase for selected items.
func (l *TodoList) SetStatusByIDs(ids []string, status TodoStatus, phase string) int {
	if len(ids) == 0 {
		return 0
	}
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return 0
	}
	l.mu.Lock()
	changed := 0
	for i, item := range l.items {
		if _, ok := idSet[item.ID]; !ok {
			continue
		}
		if item.Status == TodoStatusRunning && status != TodoStatusRunning {
			l.cancelActiveLocked(item.ID)
		}
		item.Status = status
		if strings.TrimSpace(phase) != "" {
			item.Phase = phase
		} else {
			item.Phase = string(status)
		}
		l.items[i] = item
		changed++
	}
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	return changed
}

// RefreshUnfinishedRequests updates stored requests for retryable tasks while preserving their URLs and task state.
func (l *TodoList) RefreshUnfinishedRequests(update func(tasks.BrowserLaunchRequest) tasks.BrowserLaunchRequest) int {
	if update == nil {
		return 0
	}
	l.mu.Lock()
	changed := 0
	for i, item := range l.items {
		if !todoStatusIsRestartable(item.Status) {
			continue
		}
		next := update(item.Request).Normalize()
		if strings.TrimSpace(next.URL) == "" {
			next.URL = item.Request.URL
		}
		item.Request = next
		l.items[i] = item
		changed++
	}
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil && changed > 0 {
		notifier()
	}
	return changed
}

// StartByIDs reruns the selected tasks immediately.
func (l *TodoList) StartByIDs(ids []string, runner TodoRunner) ([]TodoItem, error) {
	if runner == nil {
		runner = tasks.RunBrowserRequest
	}
	if len(ids) == 0 {
		return nil, nil
	}
	items := make([]TodoItem, 0, len(ids))
	var runErr error
	for _, id := range ids {
		item, ok := l.ItemByID(id)
		if !ok {
			err := fmt.Errorf("todo item %s not found", id)
			if runErr == nil {
				runErr = err
			} else {
				runErr = errors.Join(runErr, err)
			}
			continue
		}
		running, err := l.runExistingItem(item.ID, runner)
		items = append(items, running)
		if err != nil {
			if runErr == nil {
				runErr = err
			} else {
				runErr = errors.Join(runErr, err)
			}
		}
	}
	return items, runErr
}

// RunByIDs reruns the selected tasks in place without creating new todo rows.
func (l *TodoList) RunByIDs(ids []string, runner TodoRunner) ([]TodoItem, error) {
	if runner == nil {
		runner = tasks.RunBrowserRequest
	}
	if len(ids) == 0 {
		return nil, nil
	}
	items := make([]TodoItem, 0, len(ids))
	var runErr error
	for _, id := range ids {
		item, ok := l.ItemByID(id)
		if !ok {
			err := fmt.Errorf("todo item %s not found", id)
			if runErr == nil {
				runErr = err
			} else {
				runErr = errors.Join(runErr, err)
			}
			continue
		}
		running, err := l.runExistingItem(item.ID, runner)
		items = append(items, running)
		if err != nil {
			if runErr == nil {
				runErr = err
			} else {
				runErr = errors.Join(runErr, err)
			}
		}
	}
	return items, runErr
}

// TodoStatusFromTaskState maps the persisted task state into the UI queue state.
func TodoStatusFromTaskState(state tasks.TaskState) TodoStatus {
	switch state {
	case tasks.TaskStateQueued:
		return TodoStatusQueued
	case tasks.TaskStateRouting:
		return TodoStatusRouting
	case tasks.TaskStatePreparing:
		return TodoStatusPreparing
	case tasks.TaskStatePrepared, tasks.TaskStateRunning:
		return TodoStatusRunning
	case tasks.TaskStatePaused:
		return TodoStatusPaused
	case tasks.TaskStateWaitingVerification:
		return TodoStatusWaitingVerification
	case tasks.TaskStateVerificationCleared:
		return TodoStatusVerificationCleared
	case tasks.TaskStateCompleted:
		return TodoStatusCompleted
	case tasks.TaskStateFailed:
		return TodoStatusFailed
	default:
		return TodoStatusPending
	}
}

// TaskStateFromTodoStatus maps the UI queue state back to the persisted task state.
func TaskStateFromTodoStatus(status TodoStatus) tasks.TaskState {
	switch status {
	case TodoStatusQueued:
		return tasks.TaskStateQueued
	case TodoStatusRouting:
		return tasks.TaskStateRouting
	case TodoStatusPreparing:
		return tasks.TaskStatePreparing
	case TodoStatusRunning:
		return tasks.TaskStateRunning
	case TodoStatusPaused:
		return tasks.TaskStatePaused
	case TodoStatusWaitingVerification:
		return tasks.TaskStateWaitingVerification
	case TodoStatusVerificationCleared:
		return tasks.TaskStateVerificationCleared
	case TodoStatusCompleted:
		return tasks.TaskStateCompleted
	case TodoStatusFailed:
		return tasks.TaskStateFailed
	default:
		return tasks.TaskStatePrepared
	}
}

// BrowserRunning reports whether a browser session lock exists.
func (l *TodoList) BrowserRunning(workspaceRoot string) bool {
	return runtime.BrowserSessionLocked(runtime.NewPaths(workspaceRoot).Root)
}

// WaitForBrowserClose waits until the browser session lock disappears.
func (l *TodoList) WaitForBrowserClose(workspaceRoot string, pollInterval time.Duration) error {
	return runtime.WaitForBrowserSessionUnlock(runtime.NewPaths(workspaceRoot).Root, pollInterval)
}

// TodoRunner runs one browser request.
type TodoRunner func(tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error)

// RunImmediately appends one request as running, executes it, and updates the item in place.
func (l *TodoList) RunImmediately(req tasks.BrowserLaunchRequest, runner TodoRunner) (TodoItem, error) {
	if runner == nil {
		runner = tasks.RunBrowserRequest
	}
	req = req.Normalize()

	l.mu.Lock()
	l.seq++
	item := TodoItem{
		ID:        fmt.Sprintf("todo-%03d", l.seq),
		Request:   req,
		Status:    TodoStatusQueued,
		Progress:  0,
		Phase:     "queued",
		CreatedAt: time.Now().UTC(),
	}
	l.items = append([]TodoItem{item}, l.items...)
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}

	req.WorkerID = "ui"
	req.TaskID = item.ID
	req.Progress = l.makeProgressUpdater(item.ID)

	l.acquireRunSlot()
	defer l.releaseRunSlot()
	req, cancel := l.requestWithCancel(item.ID, req)
	defer l.unregisterActiveCancel(item.ID, cancel)
	if err := l.updateTaskStatus(item.ID, func(task *TodoItem) {
		task.Status = TodoStatusRunning
		task.Phase = "running"
		task.StartedAt = time.Now().UTC()
		task.StepMessage = ""
	}); err == nil {
		if notifier := l.notifier; notifier != nil {
			notifier()
		}
	}
	result, err := runner(req)

	l.mu.Lock()
	index, ok := l.findItemIndexByIDLocked(item.ID)
	if !ok {
		l.mu.Unlock()
		return TodoItem{}, fmt.Errorf("todo item %s not found after run", item.ID)
	}
	item = l.items[index]
	item.FinishedAt = time.Now().UTC()
	if item.Status != TodoStatusRunning {
		if err != nil {
			item.LastError = err.Error()
			item.StepMessage = err.Error()
		}
		l.items[index] = item
		l.mu.Unlock()
		if notifier != nil {
			notifier()
		}
		if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
			log.Printf("save task report failed: %v", err)
		}
		return item, err
	}
	if err != nil {
		item.Status = TodoStatusFailed
		item.Progress = 1
		item.Phase = "failed"
		item.StepMessage = err.Error()
		item.LastError = err.Error()
		l.items[index] = item
		l.mu.Unlock()
		if notifier != nil {
			notifier()
		}
		if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
			log.Printf("save task report failed: %v", err)
		}
		return item, err
	}
	item.Status = TodoStatusCompleted
	item.Progress = 1
	item.Phase = "completed"
	item.StepCurrent = item.StepTotal
	item.StepMessage = "completed"
	item.Result = result
	item.StepBytes = result.DownloadedBytes
	item.StepBPS = 0
	item.AverageBPS = result.DownloadAverageBPS
	item.LastError = ""
	l.items[index] = item
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
		log.Printf("save task report failed: %v", err)
	}
	if err := CleanupTaskLog(req.RuntimeRoot, item.ID); err != nil {
		log.Printf("cleanup task log failed: %v", err)
	}
	return item, nil
}

// StartAllUnfinishedAfterBrowserClose waits for the browser session lock to disappear, then starts all pending items.
func (l *TodoList) StartAllUnfinishedAfterBrowserClose(workspaceRoot string, pollInterval time.Duration, runner TodoRunner) ([]TodoItem, error) {
	return l.StartAllUnfinishedAfterBrowserCloseWithConcurrency(workspaceRoot, 1, pollInterval, runner)
}

// StartAllUnfinishedAfterBrowserCloseWithConcurrency waits for the browser session lock to disappear, then starts all pending items with a concurrency limit.
func (l *TodoList) StartAllUnfinishedAfterBrowserCloseWithConcurrency(workspaceRoot string, maxParallel int, pollInterval time.Duration, runner TodoRunner) ([]TodoItem, error) {
	if runner == nil {
		runner = tasks.RunBrowserRequest
	}
	runtimeRoot := runtime.NewPaths(workspaceRoot).Root
	if runtime.BrowserSessionLocked(runtimeRoot) {
		if err := runtime.WaitForBrowserSessionUnlock(runtimeRoot, pollInterval); err != nil {
			return nil, err
		}
	}
	return l.startAllPending(maxParallel, runner)
}

// StartAllUnfinished starts all pending items immediately.
func (l *TodoList) StartAllUnfinished(workspaceRoot string, runner TodoRunner) ([]TodoItem, error) {
	return l.StartAllUnfinishedWithConcurrency(workspaceRoot, 1, runner)
}

// StartAllUnfinishedWithConcurrency starts all pending items immediately using a concurrency limit.
func (l *TodoList) StartAllUnfinishedWithConcurrency(workspaceRoot string, maxParallel int, runner TodoRunner) ([]TodoItem, error) {
	if runner == nil {
		runner = tasks.RunBrowserRequest
	}
	_ = workspaceRoot
	return l.startAllPending(maxParallel, runner)
}

func (l *TodoList) startAllPending(maxParallel int, runner TodoRunner) ([]TodoItem, error) {
	l.mu.Lock()
	ids := make([]string, 0, len(l.items))
	for _, item := range l.items {
		if todoStatusIsRestartable(item.Status) {
			ids = append(ids, item.ID)
		}
	}
	l.mu.Unlock()

	results := make([]TodoItem, 0, len(ids))
	var runErr error
	if maxParallel <= 0 {
		maxParallel = 1
	}
	if maxParallel == 1 {
		for _, id := range ids {
			item, err := l.runPendingItem(id, runner)
			results = append(results, item)
			if err != nil {
				if runErr == nil {
					runErr = err
				} else {
					runErr = errors.Join(runErr, err)
				}
			}
		}
		return results, runErr
	}

	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	for _, id := range ids {
		id := id
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			item, err := l.runPendingItem(id, runner)
			resultsMu.Lock()
			results = append(results, item)
			if err != nil {
				if runErr == nil {
					runErr = err
				} else {
					runErr = errors.Join(runErr, err)
				}
			}
			resultsMu.Unlock()
		}()
	}
	wg.Wait()
	return results, runErr
}

func todoStatusIsRestartable(status TodoStatus) bool {
	switch status {
	case TodoStatusPending,
		TodoStatusQueued,
		TodoStatusRouting,
		TodoStatusPreparing,
		TodoStatusPaused,
		TodoStatusWaitingVerification,
		TodoStatusVerificationCleared,
		TodoStatusFailed:
		return true
	default:
		return false
	}
}

func todoStatusIsActive(status TodoStatus) bool {
	switch status {
	case TodoStatusRunning:
		return true
	default:
		return false
	}
}

func (l *TodoList) runPendingItem(itemID string, runner TodoRunner) (TodoItem, error) {
	l.mu.Lock()
	index, ok := l.findItemIndexByIDLocked(itemID)
	if !ok {
		l.mu.Unlock()
		return TodoItem{}, nil
	}
	item := l.items[index]
	item.Status = TodoStatusQueued
	item.Progress = 0
	item.Phase = "queued"
	item.StepCurrent = 0
	item.StepTotal = 0
	item.StepMessage = ""
	item.StepBytes = 0
	item.StepBPS = 0
	item.AverageBPS = 0
	l.items[index] = item
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}

	req := item.Request
	if strings.TrimSpace(req.WorkerID) == "" {
		req.WorkerID = "ui"
	}
	if strings.TrimSpace(req.TaskID) == "" {
		req.TaskID = item.ID
	}
	req.Progress = l.makeProgressUpdater(item.ID)

	l.acquireRunSlot()
	defer l.releaseRunSlot()
	if item, ok, err := l.itemBeforeRun(item.ID); err != nil || !ok {
		return item, err
	} else if item.Status == TodoStatusPaused {
		return item, nil
	}
	req, cancel := l.requestWithCancel(item.ID, req)
	defer l.unregisterActiveCancel(item.ID, cancel)
	if err := l.updateTaskStatus(item.ID, func(task *TodoItem) {
		task.Status = TodoStatusRunning
		task.Phase = "running"
		task.StartedAt = time.Now().UTC()
		task.StepMessage = ""
	}); err == nil {
		if notifier := l.notifier; notifier != nil {
			notifier()
		}
	}
	result, err := runner(req)

	l.mu.Lock()
	index, ok = l.findItemIndexByIDLocked(item.ID)
	if !ok {
		l.mu.Unlock()
		return TodoItem{}, fmt.Errorf("todo item %s not found after run", item.ID)
	}
	item = l.items[index]
	item.FinishedAt = time.Now().UTC()
	if item.Status != TodoStatusRunning {
		if err != nil {
			item.LastError = err.Error()
			item.StepMessage = err.Error()
		}
		l.items[index] = item
		l.mu.Unlock()
		if notifier != nil {
			notifier()
		}
		if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
			log.Printf("save task report failed: %v", err)
		}
		return item, err
	}
	if err != nil {
		item.Status = TodoStatusFailed
		item.Progress = 1
		item.Phase = "failed"
		item.StepMessage = err.Error()
		item.LastError = err.Error()
		l.items[index] = item
		l.mu.Unlock()
		if notifier != nil {
			notifier()
		}
		if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
			log.Printf("save task report failed: %v", err)
		}
		return item, err
	}
	item.Status = TodoStatusCompleted
	item.Progress = 1
	item.Phase = "completed"
	item.StepCurrent = item.StepTotal
	item.StepMessage = "completed"
	item.Result = result
	item.StepBytes = result.DownloadedBytes
	item.StepBPS = 0
	item.AverageBPS = result.DownloadAverageBPS
	item.LastError = ""
	l.items[index] = item
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
		log.Printf("save task report failed: %v", err)
	}
	if err := CleanupTaskLog(req.RuntimeRoot, item.ID); err != nil {
		log.Printf("cleanup task log failed: %v", err)
	}
	return item, nil
}

func (l *TodoList) runExistingItem(itemID string, runner TodoRunner) (TodoItem, error) {
	l.mu.Lock()
	index, ok := l.findItemIndexByIDLocked(itemID)
	if !ok {
		l.mu.Unlock()
		return TodoItem{}, fmt.Errorf("todo item %s not found", itemID)
	}
	item := l.items[index]
	item.Status = TodoStatusQueued
	item.Progress = 0
	item.Phase = "queued"
	item.StepCurrent = 0
	item.StepTotal = 0
	item.StepMessage = ""
	item.LastError = ""
	item.Result = tasks.BrowserRunResult{}
	item.StepBytes = 0
	item.StepBPS = 0
	item.AverageBPS = 0
	l.items[index] = item
	notifier := l.notifier
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}

	req := item.Request
	if strings.TrimSpace(req.WorkerID) == "" {
		req.WorkerID = "ui"
	}
	if strings.TrimSpace(req.TaskID) == "" {
		req.TaskID = item.ID
	}
	req.Progress = l.makeProgressUpdater(item.ID)

	l.acquireRunSlot()
	defer l.releaseRunSlot()
	if item, ok, err := l.itemBeforeRun(item.ID); err != nil || !ok {
		return item, err
	} else if item.Status == TodoStatusPaused {
		return item, nil
	}
	req, cancel := l.requestWithCancel(item.ID, req)
	defer l.unregisterActiveCancel(item.ID, cancel)
	if err := l.updateTaskStatus(item.ID, func(task *TodoItem) {
		task.Status = TodoStatusRunning
		task.Phase = "running"
		task.StartedAt = time.Now().UTC()
		task.StepMessage = ""
	}); err == nil {
		if notifier := l.notifier; notifier != nil {
			notifier()
		}
	}
	result, err := runner(req)

	l.mu.Lock()
	index, ok = l.findItemIndexByIDLocked(item.ID)
	if !ok {
		l.mu.Unlock()
		return TodoItem{}, fmt.Errorf("todo item %s not found after run", item.ID)
	}
	item = l.items[index]
	item.FinishedAt = time.Now().UTC()
	if item.Status != TodoStatusRunning {
		if err != nil {
			item.LastError = err.Error()
			item.StepMessage = err.Error()
		}
		l.items[index] = item
		l.mu.Unlock()
		if notifier != nil {
			notifier()
		}
		if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
			log.Printf("save task report failed: %v", err)
		}
		return item, err
	}
	if err != nil {
		item.Status = TodoStatusFailed
		item.Progress = 1
		item.Phase = "failed"
		item.StepMessage = err.Error()
		item.LastError = err.Error()
		l.items[index] = item
		l.mu.Unlock()
		if notifier != nil {
			notifier()
		}
		if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
			log.Printf("save task report failed: %v", err)
		}
		return item, err
	}
	item.Status = TodoStatusCompleted
	item.Progress = 1
	item.Phase = "completed"
	item.StepCurrent = item.StepTotal
	item.StepMessage = "completed"
	item.Result = result
	item.StepBytes = result.DownloadedBytes
	item.StepBPS = 0
	item.AverageBPS = result.DownloadAverageBPS
	item.LastError = ""
	l.items[index] = item
	l.mu.Unlock()
	if notifier != nil {
		notifier()
	}
	if err := SaveTaskReport(req.RuntimeRoot, item); err != nil {
		log.Printf("save task report failed: %v", err)
	}
	if err := CleanupTaskLog(req.RuntimeRoot, item.ID); err != nil {
		log.Printf("cleanup task log failed: %v", err)
	}
	return item, nil
}

func (l *TodoList) itemBeforeRun(itemID string) (TodoItem, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	index, ok := l.findItemIndexByIDLocked(itemID)
	if !ok {
		return TodoItem{}, false, nil
	}
	item := l.items[index]
	switch item.Status {
	case TodoStatusPaused:
		item.Phase = "paused"
		l.items[index] = item
		return item, true, nil
	case TodoStatusQueued, TodoStatusPending, TodoStatusFailed, TodoStatusWaitingVerification, TodoStatusVerificationCleared:
		return item, true, nil
	default:
		return item, true, nil
	}
}

func (l *TodoList) requestWithCancel(itemID string, req tasks.BrowserLaunchRequest) (tasks.BrowserLaunchRequest, context.CancelFunc) {
	parent := req.Context
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	req.Context = ctx
	l.mu.Lock()
	if l.activeCancels == nil {
		l.activeCancels = map[string]context.CancelFunc{}
	}
	l.activeCancels[itemID] = cancel
	l.mu.Unlock()
	return req, cancel
}

func (l *TodoList) unregisterActiveCancel(itemID string, cancel context.CancelFunc) {
	l.mu.Lock()
	delete(l.activeCancels, itemID)
	l.mu.Unlock()
	cancel()
}

func (l *TodoList) cancelActiveLocked(itemID string) {
	if l.activeCancels == nil {
		return
	}
	if cancel := l.activeCancels[itemID]; cancel != nil {
		cancel()
	}
}

func (l *TodoList) acquireRunSlot() {
	l.mu.Lock()
	if l.cond == nil {
		l.cond = sync.NewCond(&l.mu)
	}
	for l.running >= l.maxParallel {
		l.cond.Wait()
	}
	l.running++
	l.mu.Unlock()
}

func (l *TodoList) releaseRunSlot() {
	l.mu.Lock()
	if l.running > 0 {
		l.running--
	}
	if l.cond == nil {
		l.cond = sync.NewCond(&l.mu)
	}
	l.cond.Broadcast()
	l.mu.Unlock()
}

func (l *TodoList) updateTaskStatus(itemID string, fn func(*TodoItem)) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	index, ok := l.findItemIndexByIDLocked(itemID)
	if !ok {
		return fmt.Errorf("todo item %s not found", itemID)
	}
	fn(&l.items[index])
	return nil
}

func (l *TodoList) sortedItemsLocked() []TodoItem {
	items := make([]TodoItem, len(l.items))
	copy(items, l.items)
	sort.SliceStable(items, func(i, j int) bool {
		pi := todoItemSortPriority(items[i])
		pj := todoItemSortPriority(items[j])
		if pi != pj {
			return pi < pj
		}
		ni := todoItemNumber(items[i].ID)
		nj := todoItemNumber(items[j].ID)
		if ni != nj {
			return ni > nj
		}
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items
}

func todoItemSortPriority(item TodoItem) int {
	switch item.Status {
	case TodoStatusPending, TodoStatusQueued, TodoStatusRouting, TodoStatusPreparing, TodoStatusRunning, TodoStatusPaused, TodoStatusWaitingVerification, TodoStatusVerificationCleared:
		return 0
	case TodoStatusFailed:
		return 1
	case TodoStatusCompleted:
		return 2
	default:
		return 3
	}
}

func todoItemNumber(id string) int {
	return todoItemSequence(id)
}

func (l *TodoList) makeProgressUpdater(itemID string) func(zeri.DownloadProgress) {
	return func(update zeri.DownloadProgress) {
		l.mu.Lock()
		index, ok := l.findItemIndexByIDLocked(itemID)
		if !ok {
			l.mu.Unlock()
			return
		}
		item := l.items[index]
		if update.Fraction >= 0 {
			item.Progress = clamp01(update.Fraction)
		}
		if update.Total > 0 {
			item.StepCurrent = update.Current
			item.StepTotal = update.Total
		}
		if strings.TrimSpace(update.Phase) != "" {
			item.Phase = update.Phase
		}
		if strings.TrimSpace(update.Message) != "" {
			item.StepMessage = update.Message
		}
		if update.Bytes > 0 {
			item.StepBytes = update.Bytes
		}
		if update.BytesPerSecond > 0 {
			item.StepBPS = update.BytesPerSecond
		}
		if update.AverageBytesPerSecond > 0 {
			item.AverageBPS = update.AverageBytesPerSecond
		}
		l.items[index] = item
		notifier := l.notifier
		l.mu.Unlock()
		l.scheduleProgressNotify(notifier)
	}
}

func (l *TodoList) scheduleProgressNotify(notifier func()) {
	if notifier == nil {
		return
	}
	delay := l.progressDelay
	if delay <= 0 {
		delay = 80 * time.Millisecond
	}
	l.progressNotifyMu.Lock()
	l.progressNotifyAt = time.Now()
	if l.progressNotifyRun {
		l.progressNotifyMu.Unlock()
		return
	}
	l.progressNotifyRun = true
	l.progressNotifyMu.Unlock()
	time.AfterFunc(delay, func() {
		for {
			l.progressNotifyMu.Lock()
			elapsed := time.Since(l.progressNotifyAt)
			if elapsed < delay {
				remaining := delay - elapsed
				l.progressNotifyMu.Unlock()
				time.Sleep(remaining)
				continue
			}
			l.progressNotifyRun = false
			l.progressNotifyMu.Unlock()
			notifier()
			return
		}
	})
}

func (l *TodoList) findItemIndexByIDLocked(id string) (int, bool) {
	for i, item := range l.items {
		if item.ID == id {
			return i, true
		}
	}
	return -1, false
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func browserLaunchRequestKey(req tasks.BrowserLaunchRequest) string {
	req = req.Normalize()
	return strings.ToLower(strings.TrimSpace(req.BrowserType)) + "\n" + strings.TrimSpace(req.URL)
}
