package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/tasks"
)

func TestTodoListAddsPendingItems(t *testing.T) {
	list := NewTodoList()
	item := list.AddPending(tasks.BrowserLaunchRequest{
		URL:         "https://example.com",
		Headless:    true,
		BrowserPath: `C:\Program Files\Mozilla Firefox\firefox.exe`,
	})

	if item.Status != TodoStatusPending {
		t.Fatalf("item.Status = %q, want pending", item.Status)
	}
	if len(list.Pending()) != 1 {
		t.Fatalf("len(Pending()) = %d, want 1", len(list.Pending()))
	}
}

func TestTodoListRunImmediatelyCompletesItem(t *testing.T) {
	list := NewTodoList()
	runtimeRoot := t.TempDir()
	item, err := list.RunImmediately(tasks.BrowserLaunchRequest{URL: "https://example.com", RuntimeRoot: runtimeRoot}, func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
		return tasks.BrowserRunResult{URL: req.URL, Title: "ok"}, nil
	})
	if err != nil {
		t.Fatalf("RunImmediately() error = %v", err)
	}
	if item.Status != TodoStatusCompleted {
		t.Fatalf("item.Status = %q, want completed", item.Status)
	}
	if item.Result.Title != "ok" {
		t.Fatalf("item.Result.Title = %q, want ok", item.Result.Title)
	}
	if len(list.Items()) != 1 {
		t.Fatalf("len(Items()) = %d, want 1", len(list.Items()))
	}
	reportPath := filepath.Join(runtimeRoot, "tasks", "task-"+item.ID, "report.json")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("saved report not found: %v", err)
	}
}

func TestTodoListConcurrencyLimitSerializesImmediateRuns(t *testing.T) {
	list := NewTodoList()
	list.SetConcurrencyLimit(1)
	runtimeRoot := t.TempDir()

	started := make(chan string, 2)
	releaseFirst := make(chan struct{})
	runner := func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
		started <- req.TaskID
		if req.TaskID == "todo-001" {
			<-releaseFirst
		}
		return tasks.BrowserRunResult{URL: req.URL, Title: req.TaskID}, nil
	}

	done1 := make(chan error, 1)
	go func() {
		_, err := list.RunImmediately(tasks.BrowserLaunchRequest{URL: "https://one.example", RuntimeRoot: runtimeRoot}, runner)
		done1 <- err
	}()

	select {
	case id := <-started:
		if id != "todo-001" {
			t.Fatalf("first started task = %q, want todo-001", id)
		}
	case <-time.After(time.Second):
		t.Fatal("first task did not start")
	}

	done2 := make(chan error, 1)
	go func() {
		_, err := list.RunImmediately(tasks.BrowserLaunchRequest{URL: "https://two.example", RuntimeRoot: runtimeRoot}, runner)
		done2 <- err
	}()

	select {
	case id := <-started:
		t.Fatalf("second task started early: %q", id)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)

	select {
	case id := <-started:
		if id != "todo-002" {
			t.Fatalf("second started task = %q, want todo-002", id)
		}
	case <-time.After(time.Second):
		t.Fatal("second task did not start after slot was released")
	}

	if err := <-done1; err != nil {
		t.Fatalf("first task error = %v", err)
	}
	if err := <-done2; err != nil {
		t.Fatalf("second task error = %v", err)
	}
}

func TestTodoListWaitsForBrowserLockBeforeStarting(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(workspace, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))
	runtimeRoot := runtime.NewPaths(workspace).Root

	list := NewTodoList()
	list.AddPending(tasks.BrowserLaunchRequest{URL: "https://example.com"})

	release, err := runtime.AcquireBrowserSessionLock(runtimeRoot)
	if err != nil {
		t.Fatalf("AcquireBrowserSessionLock() error = %v", err)
	}

	done := make(chan []TodoItem, 1)
	runErr := make(chan error, 1)
	go func() {
		items, err := list.StartAllUnfinishedAfterBrowserClose(workspace, 10*time.Millisecond, func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
			return tasks.BrowserRunResult{URL: req.URL, Title: "ok"}, nil
		})
		done <- items
		runErr <- err
	}()

	select {
	case <-done:
		t.Fatal("start returned before browser lock was released")
	case <-time.After(50 * time.Millisecond):
	}

	if err := release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	var items []TodoItem
	select {
	case items = <-done:
	case <-time.After(time.Second):
		t.Fatal("start did not finish after lock release")
	}
	if err := <-runErr; err != nil {
		t.Fatalf("StartAllUnfinishedAfterBrowserClose() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Status != TodoStatusCompleted {
		t.Fatalf("items[0].Status = %q, want completed", items[0].Status)
	}
	if items[0].Result.Title != "ok" {
		t.Fatalf("items[0].Result.Title = %q, want ok", items[0].Result.Title)
	}
}

func TestTodoListStartAllDoesNotCheckBrowserLock(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(workspace, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))
	runtimeRoot := runtime.NewPaths(workspace).Root

	list := NewTodoList()
	list.AddPending(tasks.BrowserLaunchRequest{URL: "https://example.com"})

	release, err := runtime.AcquireBrowserSessionLock(runtimeRoot)
	if err != nil {
		t.Fatalf("AcquireBrowserSessionLock() error = %v", err)
	}
	defer func() {
		_ = release()
	}()

	items, err := list.StartAllUnfinished(workspace, func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
		return tasks.BrowserRunResult{URL: req.URL, Title: "ok"}, nil
	})
	if err != nil {
		t.Fatalf("StartAllUnfinished() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Status != TodoStatusCompleted {
		t.Fatalf("items[0].Status = %q, want completed", items[0].Status)
	}
}

func TestTodoListClearFinishedRemovesCompletedItems(t *testing.T) {
	list := NewTodoList()
	list.AddPending(tasks.BrowserLaunchRequest{URL: "https://example.com"})
	list.AddPending(tasks.BrowserLaunchRequest{URL: "https://example.org"})

	list.mu.Lock()
	list.items[0].Status = TodoStatusCompleted
	list.items[1].Status = TodoStatusFailed
	list.mu.Unlock()

	removed := list.ClearFinished()
	if removed != 2 {
		t.Fatalf("ClearFinished() removed = %d, want 2", removed)
	}
	if len(list.Items()) != 0 {
		t.Fatalf("len(Items()) = %d, want 0", len(list.Items()))
	}
}

func TestTodoListRemoveByIDsKeepsActiveTasks(t *testing.T) {
	list := NewTodoList()
	list.mu.Lock()
	list.items = []TodoItem{
		{ID: "todo-running", Status: TodoStatusRunning},
		{ID: "todo-queued", Status: TodoStatusQueued},
		{ID: "todo-preparing", Status: TodoStatusPreparing},
		{ID: "todo-paused", Status: TodoStatusPaused},
		{ID: "todo-failed", Status: TodoStatusFailed},
	}
	list.mu.Unlock()

	removed := list.RemoveByIDs([]string{"todo-running", "todo-queued", "todo-preparing", "todo-paused", "todo-failed"})
	if removed != 4 {
		t.Fatalf("removed = %d, want 4", removed)
	}
	if _, ok := list.ItemByID("todo-running"); !ok {
		t.Fatal("running task was removed")
	}
	if _, ok := list.ItemByID("todo-queued"); ok {
		t.Fatal("queued task was not removed")
	}
	if _, ok := list.ItemByID("todo-preparing"); ok {
		t.Fatal("preparing task was not removed")
	}
	if _, ok := list.ItemByID("todo-paused"); ok {
		t.Fatal("paused task was not removed")
	}
	if _, ok := list.ItemByID("todo-failed"); ok {
		t.Fatal("failed task was not removed")
	}
}

func TestTodoListPauseUpdatesRunningTasks(t *testing.T) {
	list := NewTodoList()
	list.mu.Lock()
	list.items = []TodoItem{
		{ID: "todo-running", Status: TodoStatusRunning},
		{ID: "todo-queued", Status: TodoStatusQueued},
	}
	list.mu.Unlock()

	changed := list.SetStatusByIDs([]string{"todo-running", "todo-queued"}, TodoStatusPaused, "paused")
	if changed != 2 {
		t.Fatalf("changed = %d, want 2", changed)
	}
	running, ok := list.ItemByID("todo-running")
	if !ok || running.Status != TodoStatusPaused {
		t.Fatalf("running status = %q ok=%v, want paused", running.Status, ok)
	}
	queued, ok := list.ItemByID("todo-queued")
	if !ok || queued.Status != TodoStatusPaused {
		t.Fatalf("queued status = %q ok=%v, want paused", queued.Status, ok)
	}
}

func TestTodoListStatusChangeCancelsRunningTask(t *testing.T) {
	list := NewTodoList()
	started := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		_, err := list.RunImmediately(tasks.BrowserLaunchRequest{URL: "https://example.com"}, func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
			started <- req.TaskID
			<-req.Context.Done()
			return tasks.BrowserRunResult{}, req.Context.Err()
		})
		done <- err
	}()

	var id string
	select {
	case id = <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}
	changed := list.SetStatusByIDs([]string{id}, TodoStatusPaused, "paused")
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("RunImmediately() error = nil, want cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("task did not stop after status change")
	}
	item, ok := list.ItemByID(id)
	if !ok {
		t.Fatalf("task %s missing", id)
	}
	if item.Status != TodoStatusPaused {
		t.Fatalf("task status = %q, want paused", item.Status)
	}
}

func TestTodoListRefreshUnfinishedRequests(t *testing.T) {
	list := NewTodoList()
	list.mu.Lock()
	list.items = []TodoItem{
		{
			ID:     "todo-1",
			Status: TodoStatusFailed,
			Request: tasks.BrowserLaunchRequest{
				URL:         "https://failed.example",
				BrowserPath: `C:\old\firefox.exe`,
				DriverDir:   `C:\old\driver`,
			},
		},
		{
			ID:     "todo-2",
			Status: TodoStatusCompleted,
			Request: tasks.BrowserLaunchRequest{
				URL:         "https://done.example",
				BrowserPath: `C:\old\firefox.exe`,
				DriverDir:   `C:\old\driver`,
			},
		},
	}
	list.mu.Unlock()

	changed := list.RefreshUnfinishedRequests(func(req tasks.BrowserLaunchRequest) tasks.BrowserLaunchRequest {
		req.BrowserPath = `D:\new\firefox.exe`
		req.DriverDir = `D:\new\driver`
		return req
	})
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}

	failed, ok := list.ItemByID("todo-1")
	if !ok {
		t.Fatal("failed task missing")
	}
	if failed.Request.BrowserPath != `D:\new\firefox.exe` || failed.Request.DriverDir != `D:\new\driver` {
		t.Fatalf("failed request not refreshed: %+v", failed.Request)
	}
	done, ok := list.ItemByID("todo-2")
	if !ok {
		t.Fatal("completed task missing")
	}
	if done.Request.BrowserPath != `C:\old\firefox.exe` || done.Request.DriverDir != `C:\old\driver` {
		t.Fatalf("completed request changed unexpectedly: %+v", done.Request)
	}
}

func TestTodoListSortsRunningFailedCompletedByIDDesc(t *testing.T) {
	list := NewTodoList()
	list.mu.Lock()
	list.items = []TodoItem{
		{ID: "todo-10", Status: TodoStatusCompleted, CreatedAt: time.Unix(10, 0).UTC()},
		{ID: "todo-2", Status: TodoStatusFailed, CreatedAt: time.Unix(20, 0).UTC()},
		{ID: "todo-7", Status: TodoStatusRunning, CreatedAt: time.Unix(30, 0).UTC()},
		{ID: "todo-5", Status: TodoStatusCompleted, CreatedAt: time.Unix(40, 0).UTC()},
		{ID: "todo-8", Status: TodoStatusFailed, CreatedAt: time.Unix(50, 0).UTC()},
		{ID: "todo-3", Status: TodoStatusRunning, CreatedAt: time.Unix(60, 0).UTC()},
	}
	list.mu.Unlock()

	items := list.Items()
	want := []string{"todo-7", "todo-3", "todo-8", "todo-2", "todo-10", "todo-5"}
	if len(items) != len(want) {
		t.Fatalf("len(items) = %d, want %d", len(items), len(want))
	}
	for i, item := range items {
		if item.ID != want[i] {
			t.Fatalf("items[%d].ID = %q, want %q", i, item.ID, want[i])
		}
	}
}

func TestTodoListStartAllIncludesRestartableStatuses(t *testing.T) {
	list := NewTodoList()
	workspace := t.TempDir()
	runtimeRoot := filepath.Join(workspace, "runtime")
	list.mu.Lock()
	list.items = []TodoItem{
		{ID: "todo-1", Status: TodoStatusPaused, Request: tasks.BrowserLaunchRequest{URL: "https://paused.example", RuntimeRoot: runtimeRoot}},
		{ID: "todo-2", Status: TodoStatusFailed, Request: tasks.BrowserLaunchRequest{URL: "https://failed.example", RuntimeRoot: runtimeRoot}},
		{ID: "todo-3", Status: TodoStatusPending, Request: tasks.BrowserLaunchRequest{URL: "https://pending.example", RuntimeRoot: runtimeRoot}},
		{ID: "todo-4", Status: TodoStatusCompleted, Request: tasks.BrowserLaunchRequest{URL: "https://done.example", RuntimeRoot: runtimeRoot}},
	}
	list.mu.Unlock()

	items, err := list.StartAllUnfinished(workspace, func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
		return tasks.BrowserRunResult{URL: req.URL, Title: req.TaskID}, nil
	})
	if err != nil {
		t.Fatalf("StartAllUnfinished() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if item, ok := list.ItemByID("todo-4"); !ok || item.Status != TodoStatusCompleted {
		t.Fatalf("completed item changed unexpectedly: ok=%v status=%v", ok, item.Status)
	}
	for _, id := range []string{"todo-1", "todo-2", "todo-3"} {
		item, ok := list.ItemByID(id)
		if !ok {
			t.Fatalf("missing item %s", id)
		}
		if item.Status != TodoStatusCompleted {
			t.Fatalf("item %s status = %q, want completed", id, item.Status)
		}
	}
}
