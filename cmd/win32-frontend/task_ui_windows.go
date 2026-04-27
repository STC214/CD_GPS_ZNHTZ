//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"comic_downloader_go_playwright_stealth/ui"
)

const (
	taskContextMenuRetry   = menuIDTaskRetry
	taskContextMenuDetails = menuIDTaskDetails
	taskContextMenuOpenDir = menuIDTaskOpenDir
	taskContextMenuCopyURL = menuIDTaskCopyURL
	taskContextMenuDelete  = menuIDTaskDelete
	taskContextMenuStart   = menuIDTaskStart
	taskContextMenuPause   = menuIDTaskPause

	taskContextPopupFlags = TPM_LEFTALIGN | TPM_RIGHTBUTTON | TPM_RETURNCMD | TPM_NONOTIFY
)

var (
	promptWndProc = syscall.NewCallback(concurrencyPromptWindowProc)

	promptStateMu sync.Mutex
	promptStates  = map[HWND]*concurrencyPromptState{}
	promptOnce    sync.Once
	promptClass   = "ConcurrencyPromptDialog"

	procEnableWindow     = user32.NewProc("EnableWindow")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
)

type concurrencyPromptState struct {
	owner       HWND
	hwnd        HWND
	edit        HWND
	value       int
	result      int
	ok          bool
	done        chan struct{}
	windowTitle string
	labelText   string
}

func registerConcurrencyPromptClass(hInstance HINSTANCE) error {
	var err error
	promptOnce.Do(func() {
		className, convErr := utf16Ptr(promptClass)
		if convErr != nil {
			err = convErr
			return
		}
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
		wc := WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
			LpfnWndProc:   promptWndProc,
			HInstance:     hInstance,
			HCursor:       HCURSOR(cursor),
			HbrBackground: darkBgBrush,
			LpszClassName: className,
		}
		atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			err = fmt.Errorf("RegisterClassExW prompt: %w", callErr)
		}
	})
	return err
}

func promptIntegerDialog(owner HWND, current int, windowTitle, labelText string) (int, bool) {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	if err := registerConcurrencyPromptClass(HINSTANCE(hInstance)); err != nil {
		return 0, false
	}
	state := &concurrencyPromptState{
		owner:       owner,
		value:       current,
		done:        make(chan struct{}),
		windowTitle: windowTitle,
		labelText:   labelText,
	}
	className, _ := utf16Ptr(promptClass)
	title, _ := utf16Ptr(windowTitle)
	hwnd, _, err := procCreateWindowExW.Call(
		WS_EX_DLGMODALFRAME,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		360,
		180,
		uintptr(owner),
		0,
		hInstance,
		uintptr(unsafe.Pointer(state)),
	)
	if hwnd == 0 {
		log.Printf("create concurrency prompt failed: %v", err)
		return 0, false
	}
	state.hwnd = HWND(hwnd)
	if owner != 0 {
		procEnableWindow.Call(uintptr(owner), 0)
		defer procEnableWindow.Call(uintptr(owner), 1)
	}
	for {
		select {
		case <-state.done:
			if state.ok {
				return state.result, true
			}
			return 0, false
		default:
		}
		var msg MSG
		r, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(r) {
		case -1:
			log.Printf("concurrency prompt message loop failed: %v", err)
			return 0, false
		case 0:
			return 0, false
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func concurrencyPromptWindowProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		state := (*concurrencyPromptState)(unsafe.Pointer(((*CREATESTRUCT)(unsafe.Pointer(lParam))).CreateParams))
		if state == nil {
			return 0
		}
		state.hwnd = hwnd
		promptStateMu.Lock()
		promptStates[hwnd] = state
		promptStateMu.Unlock()
		createConcurrencyPromptControls(hwnd, state)
		return 0
	case WM_COMMAND:
		id := uint16(wParam & 0xFFFF)
		promptStateMu.Lock()
		state := promptStates[hwnd]
		promptStateMu.Unlock()
		if state == nil {
			break
		}
		switch id {
		case 1:
			text := strings.TrimSpace(getControlText(state.edit))
			if text == "" {
				msgBox(hwnd, "\u8bf7\u8f93\u5165\u5e76\u53d1\u6570", "\u63d0\u793a")
				return 0
			}
			n, err := strconv.Atoi(text)
			if err != nil || n <= 0 {
				msgBox(hwnd, "\u5e76\u53d1\u6570\u5fc5\u987b\u662f\u6b63\u6574\u6570", "\u63d0\u793a")
				return 0
			}
			state.result = n
			state.ok = true
			procDestroyWindow.Call(uintptr(hwnd))
			return 0
		case 2:
			state.ok = false
			procDestroyWindow.Call(uintptr(hwnd))
			return 0
		}
	case WM_CLOSE:
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		promptStateMu.Lock()
		state := promptStates[hwnd]
		delete(promptStates, hwnd)
		promptStateMu.Unlock()
		if state != nil && state.done != nil {
			close(state.done)
		}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func createConcurrencyPromptControls(hwnd HWND, state *concurrencyPromptState) {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	labels := []struct {
		class string
		text  string
		x, y  int32
		w, h  int32
		id    int
	}{
		{class: "Static", text: state.labelText, x: 18, y: 18, w: 300, h: 20, id: 1001},
		{class: "Edit", text: strconv.Itoa(max(1, state.value)), x: 18, y: 44, w: 304, h: 28, id: 1002},
		{class: "Button", text: "\u786e\u5b9a", x: 138, y: 84, w: 80, h: 28, id: 1},
		{class: "Button", text: "\u53d6\u6d88", x: 232, y: 84, w: 80, h: 28, id: 2},
	}
	for _, c := range labels {
		className, _ := utf16Ptr(c.class)
		text, _ := utf16Ptr(c.text)
		hwndChild, _, err := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(text)),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER,
			uintptr(c.x),
			uintptr(c.y),
			uintptr(c.w),
			uintptr(c.h),
			uintptr(hwnd),
			uintptr(c.id),
			hInstance,
			0,
		)
		if hwndChild == 0 {
			log.Printf("create concurrency prompt control %s failed: %v", c.class, err)
			continue
		}
		setControlFont(HWND(hwndChild), uiFont)
		if c.id == 1002 {
			state.edit = HWND(hwndChild)
		}
	}
}

func (a *frontendApp) promptConcurrency() {
	value, ok := promptIntegerDialog(a.hwnd, a.currentConcurrency(), "\u8bbe\u7f6e\u5e76\u53d1\u6570", "\u8bf7\u8f93\u5165\u5e76\u53d1\u6570\uff1a")
	if !ok {
		a.setStatus("concurrency unchanged")
		return
	}
	a.mu.Lock()
	a.concurrency = value
	a.mu.Unlock()
	a.todo.SetConcurrencyLimit(value)
	a.updateConcurrencyButton(value)
	a.persistFrontendState()
	a.setStatus("concurrency set to %d", value)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) promptProgressDelayDialog() {
	value, ok := promptIntegerDialog(a.hwnd, a.currentProgressDelayMS(), "\u8bbe\u7f6e\u8fdb\u5ea6\u5237\u65b0\u95f4\u9694", "\u8bf7\u8f93\u5165\u8fdb\u5ea6\u5237\u65b0\u95f4\u9694\uff08ms\uff09\uff1a")
	if !ok {
		a.setStatus("progress refresh interval unchanged")
		return
	}
	if value < 10 {
		value = 10
	}
	delay := time.Duration(value) * time.Millisecond
	a.mu.Lock()
	a.progressDelay = delay
	a.mu.Unlock()
	a.todo.SetProgressNotifyDelay(delay)
	a.persistFrontendState()
	a.setStatus("progress refresh interval set to %dms", value)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) taskSelected(id string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.selectedTaskIDs[id]
}

func (a *frontendApp) selectedTaskIDsSnapshot() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ids := make([]string, 0, len(a.selectedTaskIDs))
	for id := range a.selectedTaskIDs {
		ids = append(ids, id)
	}
	return ids
}

func (a *frontendApp) setTaskSelection(ids []string) {
	a.mu.Lock()
	a.selectedTaskIDs = make(map[string]bool, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			a.selectedTaskIDs[id] = true
		}
	}
	a.mu.Unlock()
	a.post(msgRefreshTasks)
}

func (a *frontendApp) clearTaskSelection() {
	a.setTaskSelection(nil)
}

func (a *frontendApp) toggleTaskSelection(id string) {
	a.mu.Lock()
	if a.selectedTaskIDs == nil {
		a.selectedTaskIDs = map[string]bool{}
	}
	if a.selectedTaskIDs[id] {
		delete(a.selectedTaskIDs, id)
	} else {
		a.selectedTaskIDs[id] = true
	}
	a.mu.Unlock()
	a.post(msgRefreshTasks)
}

func (a *frontendApp) selectSingleTask(id string) {
	if strings.TrimSpace(id) == "" {
		a.clearTaskSelection()
		return
	}
	a.setTaskSelection([]string{id})
}

func (a *frontendApp) showTaskContextMenu(hwnd HWND, screenX, screenY int) {
	selected := a.selectedTaskIDsSnapshot()
	if len(selected) == 0 {
		return
	}
	menu, _, _ := procCreatePopupMenu.Call()
	append := func(id uintptr, title string) {
		addMenuItem(menu, id, title)
	}
	append(taskContextMenuRetry, "\u91cd\u8bd5")
	append(taskContextMenuDetails, "\u8be6\u60c5")
	append(taskContextMenuOpenDir, "\u6253\u5f00\u4e0b\u8f7d\u76ee\u5f55")
	append(taskContextMenuCopyURL, "\u590d\u5236\u4efb\u52a1URL")
	append(taskContextMenuDelete, "\u5220\u9664")
	append(taskContextMenuStart, "\u5f00\u59cb")
	append(taskContextMenuPause, "\u6682\u505c")
	cmd, _, _ := procTrackPopupMenu.Call(menu, taskContextPopupFlags, uintptr(screenX), uintptr(screenY), 0, uintptr(hwnd), 0)
	procDestroyMenu.Call(menu)
	if cmd == 0 {
		return
	}
	a.handleTaskContextCommand(uintptr(cmd))
}

func (a *frontendApp) handleTaskContextCommand(cmd uintptr) {
	switch cmd {
	case taskContextMenuRetry:
		a.retrySelectedTasks()
	case taskContextMenuDetails:
		a.showSelectedTaskDetails()
	case taskContextMenuOpenDir:
		a.openSelectedTaskDownloadDir()
	case taskContextMenuCopyURL:
		a.copySelectedTaskURLs()
	case taskContextMenuDelete:
		a.deleteSelectedTasks()
	case taskContextMenuStart:
		a.startSelectedTasks()
	case taskContextMenuPause:
		a.pauseSelectedTasks()
	}
}

func (a *frontendApp) taskIDsFromSelection() []string {
	ids := a.selectedTaskIDsSnapshot()
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func (a *frontendApp) selectedTaskItems() []ui.TodoItem {
	ids := a.taskIDsFromSelection()
	if len(ids) == 0 {
		return nil
	}
	items := a.todo.Items()
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	result := make([]ui.TodoItem, 0, len(ids))
	for _, item := range items {
		if _, ok := idSet[item.ID]; ok {
			result = append(result, item)
		}
	}
	return result
}

func (a *frontendApp) retrySelectedTasks() {
	items := a.selectedTaskItems()
	if len(items) == 0 {
		return
	}
	go func() {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		if _, err := a.todo.RunByIDs(ids, nil); err != nil {
			a.setStatus("retry selected tasks failed: %v", err)
			return
		}
		a.post(msgRefreshTasks)
		a.post(msgRefreshInfo)
	}()
}

func (a *frontendApp) showSelectedTaskDetails() {
	items := a.selectedTaskItems()
	if len(items) == 0 {
		return
	}
	item := items[0]
	reportPath := a.resolveTaskReportPath(item.ID)
	details, err := ui.LoadTaskDetails(reportPath)
	if err != nil {
		details = ui.TaskDetailsFromItem(item, reportPath)
		msg := fmt.Sprintf(
			"Task: %s\r\nTitle: %s\r\nSite: %s\r\nState: %s\r\nURL: %s\r\nOutput: %s\r\nDownloadedDir: %s\r\nThumbnail: %s",
			details.TaskID,
			details.Title,
			details.Site,
			details.State,
			details.PrimaryURL,
			details.OutputRoot,
			details.DownloadRoot,
			details.ThumbnailPath,
		)
		msgBox(a.hwnd, msg, "Task Details")
		return
	}
	msg := fmt.Sprintf("Task: %s\r\nTitle: %s\r\nSite: %s\r\nState: %s\r\nURL: %s\r\nOutput: %s\r\nReport: %s\r\nDownloadRoot: %s\r\nAssets: %d",
		details.TaskID, details.Title, details.Site, details.State, details.PrimaryURL, details.OutputRoot, details.ReportPath, details.DownloadRoot, details.AssetCount)
	msgBox(a.hwnd, msg, "Task Details")
}

func (a *frontendApp) resolveTaskReportPath(taskID string) string {
	primary := a.paths.TaskReportPath(taskID)
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	legacy := filepath.Join(a.paths.TasksRoot, taskID, "report.json")
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return primary
}

func (a *frontendApp) openSelectedTaskDownloadDir() {
	items := a.selectedTaskItems()
	if len(items) == 0 {
		return
	}
	dir := strings.TrimSpace(items[0].Result.DownloadedDir)
	if dir == "" {
		dir = strings.TrimSpace(items[0].Request.OutputDir)
	}
	if dir == "" {
		return
	}
	openVerb := utf16PtrMust("open")
	dirPtr := utf16PtrMust(dir)
	_, _, _ = procShellExecuteW.Call(0, uintptr(unsafe.Pointer(openVerb)), uintptr(unsafe.Pointer(dirPtr)), 0, 0, SW_SHOWNORMAL)
}

func (a *frontendApp) copySelectedTaskURLs() {
	items := a.selectedTaskItems()
	if len(items) == 0 {
		return
	}
	urls := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Request.URL) != "" {
			urls = append(urls, item.Request.URL)
		}
	}
	if len(urls) == 0 {
		return
	}
	if err := setClipboardText(strings.Join(urls, "\r\n")); err != nil {
		a.setStatus("copy task URL failed: %v", err)
		return
	}
	a.setStatus("copied %d task URL(s)", len(urls))
}

func (a *frontendApp) deleteSelectedTasks() {
	ids := a.taskIDsFromSelection()
	if len(ids) == 0 {
		return
	}
	removed := a.todo.RemoveByIDs(ids)
	a.clearTaskSelection()
	a.setStatus("removed %d task(s)", removed)
	a.post(msgRefreshTasks)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) startSelectedTasks() {
	ids := a.taskIDsFromSelection()
	if len(ids) == 0 {
		return
	}
	go func() {
		_, err := a.todo.RunByIDs(ids, nil)
		if err != nil {
			a.setStatus("start selected tasks failed: %v", err)
			return
		}
		a.post(msgRefreshTasks)
		a.post(msgRefreshInfo)
	}()
}

func (a *frontendApp) pauseSelectedTasks() {
	ids := a.taskIDsFromSelection()
	if len(ids) == 0 {
		return
	}
	count := a.todo.SetStatusByIDs(ids, ui.TodoStatusPaused, "paused")
	a.setStatus("paused %d task(s)", count)
	a.post(msgRefreshTasks)
}

func setClipboardText(text string) error {
	encoded := append(utf16.Encode([]rune(text)), 0)
	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	size := uintptr(len(encoded) * 2)
	hMem, _, err := procGlobalAlloc.Call(GMEM_MOVEABLE, size)
	if hMem == 0 {
		return fmt.Errorf("GlobalAlloc failed: %w", err)
	}
	lock, _, _ := procGlobalLock.Call(hMem)
	if lock == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("GlobalLock failed")
	}
	buf := (*[1 << 28]uint16)(unsafe.Pointer(lock))[:len(encoded):len(encoded)]
	copy(buf, encoded)
	procGlobalUnlock.Call(hMem)
	if r, _, _ := procSetClipboardData.Call(CF_UNICODETEXT, hMem); r == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("SetClipboardData failed")
	}
	return nil
}
