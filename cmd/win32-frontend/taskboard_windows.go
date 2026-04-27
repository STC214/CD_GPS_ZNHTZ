package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"comic_downloader_go_playwright_stealth/ui"
)

const taskBoardClassName = "TaskBoardControl"

var taskBoardWndProc = syscall.NewCallback(taskBoardWindowProc)

var siteFilterOptions = []struct {
	Key   string
	Label string
}{
	{Key: "", Label: "\u663e\u793a\u5168\u90e8"},
	{Key: "zeri", Label: "Zeri"},
	{Key: "nyahentai", Label: "Nyahentai"},
	{Key: "hentai2", Label: "Hentai2"},
	{Key: "hentaiaz", Label: "Hentaiaz"},
	{Key: "hitomi", Label: "Hitomi"},
}

func registerTaskBoardClass(hInstance HINSTANCE) error {
	className, err := utf16Ptr(taskBoardClassName)
	if err != nil {
		return err
	}
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   taskBoardWndProc,
		HInstance:     hInstance,
		HCursor:       HCURSOR(cursor),
		HbrBackground: darkBgBrush,
		LpszClassName: className,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW task board: %w", err)
	}
	return nil
}

func taskBoardRefresh(hwnd HWND) {
	if app == nil {
		return
	}
	itemCount := len(app.filteredTaskItems())
	contentHeight := 0
	if itemCount > 0 {
		contentHeight = taskCardTopMargin + itemCount*(taskCardHeight+taskCardGap)
	}
	visibleHeight := taskBoardVisibleHeight(hwnd)
	app.mu.Lock()
	app.taskContentH = contentHeight
	maxScroll := contentHeight - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if app.taskScrollY > maxScroll {
		app.taskScrollY = maxScroll
	}
	scrollY := app.taskScrollY
	app.mu.Unlock()
	procSetScrollRange.Call(uintptr(hwnd), 1, 0, uintptr(maxScroll), 1)
	procSetScrollPos.Call(uintptr(hwnd), 1, uintptr(scrollY), 1)
	procInvalidateRect.Call(uintptr(hwnd), 0, 0)
}

func taskBoardVisibleHeight(hwnd HWND) int {
	var rc RECT
	if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return 0
	}
	return int(rc.Bottom - rc.Top)
}

func taskBoardWindowProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		taskBoardRefresh(hwnd)
		return 0
	case WM_SIZE:
		taskBoardRefresh(hwnd)
		return 0
	case WM_LBUTTONDOWN:
		taskBoardLeftClick(hwnd, lParam)
		return 0
	case WM_RBUTTONUP:
		taskBoardRightClick(hwnd, lParam)
		return 0
	case WM_CONTEXTMENU:
		taskBoardContextMenu(hwnd, lParam)
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_MOUSEWHEEL:
		return taskBoardMouseWheel(hwnd, wParam)
	case WM_VSCROLL:
		return taskBoardVScroll(hwnd, wParam)
	case WM_PAINT:
		taskBoardPaint(hwnd)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func taskBoardLeftClick(hwnd HWND, lParam uintptr) {
	if app == nil {
		return
	}
	x := int(int16(lParam & 0xFFFF))
	y := int(int16((lParam >> 16) & 0xFFFF))
	item, _, ok := taskBoardItemAtPoint(hwnd, x, y)
	ctrlPressed := isControlKeyPressed()
	if !ok {
		if !ctrlPressed {
			app.clearTaskSelection()
		}
		return
	}
	if ctrlPressed {
		app.toggleTaskSelection(item.ID)
		return
	}
	app.selectSingleTask(item.ID)
}

func taskBoardRightClick(hwnd HWND, lParam uintptr) {
	if app == nil {
		return
	}
	x := int(int16(lParam & 0xFFFF))
	y := int(int16((lParam >> 16) & 0xFFFF))
	item, _, ok := taskBoardItemAtPoint(hwnd, x, y)
	if ok && !app.taskSelected(item.ID) {
		app.selectSingleTask(item.ID)
	}
	if itemIDs := app.taskIDsFromSelection(); len(itemIDs) == 0 {
		return
	}
	var pt POINT
	if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		pt.X = int32(x)
		pt.Y = int32(y)
	}
	app.showTaskContextMenu(hwnd, int(pt.X), int(pt.Y))
}

func taskBoardContextMenu(hwnd HWND, lParam uintptr) {
	if app == nil {
		return
	}
	if ids := app.taskIDsFromSelection(); len(ids) == 0 {
		return
	}
	var pt POINT
	if lParam != ^uintptr(0) {
		pt.X = int32(int32(lParam & 0xFFFF))
		pt.Y = int32(int32((lParam >> 16) & 0xFFFF))
	} else if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		return
	}
	app.showTaskContextMenu(hwnd, int(pt.X), int(pt.Y))
}

func taskBoardItemAtPoint(hwnd HWND, x, y int) (ui.TodoItem, int, bool) {
	if app == nil {
		return ui.TodoItem{}, -1, false
	}
	var rc RECT
	if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return ui.TodoItem{}, -1, false
	}
	scrollY := app.taskScrollSnapshot()
	rowHeight := taskCardHeight + taskCardGap
	if rowHeight <= 0 {
		rowHeight = 1
	}
	absoluteY := y + scrollY
	if absoluteY < taskCardTopMargin {
		return ui.TodoItem{}, -1, false
	}
	index := (absoluteY - taskCardTopMargin) / rowHeight
	items := app.filteredTaskItems()
	if index < 0 || index >= len(items) {
		return ui.TodoItem{}, -1, false
	}
	top := taskCardTopMargin + index*rowHeight - scrollY
	if y < top || y > top+taskCardHeight {
		return ui.TodoItem{}, -1, false
	}
	return items[index], index, true
}

func isControlKeyPressed() bool {
	r, _, _ := procGetKeyState.Call(uintptr(VK_CONTROL))
	return r&0x8000 != 0
}

func taskBoardMouseWheel(hwnd HWND, wParam uintptr) uintptr {
	if app == nil {
		return 0
	}
	delta := int(int16((wParam >> 16) & 0xFFFF))
	if delta == 0 {
		return 0
	}
	lines := delta / 120
	if lines == 0 {
		if delta > 0 {
			lines = 1
		} else {
			lines = -1
		}
	}
	app.mu.Lock()
	scrollY := app.taskScrollY - lines*48
	maxScroll := app.taskContentH - taskBoardVisibleHeight(hwnd)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY < 0 {
		scrollY = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	app.taskScrollY = scrollY
	app.mu.Unlock()
	procSetScrollPos.Call(uintptr(hwnd), 1, uintptr(scrollY), 1)
	procInvalidateRect.Call(uintptr(hwnd), 0, 0)
	return 0
}

func taskBoardVScroll(hwnd HWND, wParam uintptr) uintptr {
	if app == nil {
		return 0
	}
	code := int(wParam & 0xFFFF)
	thumbPos := int((wParam >> 16) & 0xFFFF)
	app.mu.Lock()
	scrollY := app.taskScrollY
	maxScroll := app.taskContentH - taskBoardVisibleHeight(hwnd)
	if maxScroll < 0 {
		maxScroll = 0
	}
	page := taskBoardVisibleHeight(hwnd) / 2
	if page < 40 {
		page = 40
	}
	switch code {
	case SB_LINEUP:
		scrollY -= 32
	case SB_LINEDOWN:
		scrollY += 32
	case SB_PAGEUP:
		scrollY -= page
	case SB_PAGEDOWN:
		scrollY += page
	case SB_THUMBPOSITION, SB_THUMBTRACK:
		scrollY = thumbPos
	case SB_TOP:
		scrollY = 0
	case SB_BOTTOM:
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	app.taskScrollY = scrollY
	app.mu.Unlock()
	procSetScrollPos.Call(uintptr(hwnd), 1, uintptr(scrollY), 1)
	procInvalidateRect.Call(uintptr(hwnd), 0, 0)
	return 0
}

func taskBoardPaint(hwnd HWND) {
	var ps PAINTSTRUCT
	paintDC, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	if paintDC == 0 {
		return
	}
	defer procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	var rc RECT
	if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return
	}
	width := int(rc.Right - rc.Left)
	height := int(rc.Bottom - rc.Top)
	if width <= 0 || height <= 0 {
		return
	}

	memDC, _, _ := procCreateCompatibleDC.Call(paintDC)
	if memDC == 0 {
		return
	}
	defer procDeleteDC.Call(memDC)

	memBmp, _, _ := procCreateCompatibleBitmap.Call(paintDC, uintptr(width), uintptr(height))
	if memBmp == 0 {
		return
	}
	defer procDeleteObject.Call(memBmp)

	oldBmp, _, _ := procSelectObject.Call(memDC, memBmp)
	if oldBmp == 0 {
		return
	}
	defer procSelectObject.Call(memDC, oldBmp)

	brushBg, _, _ := procCreateSolidBrush.Call(uintptr(rgb(18, 22, 28)))
	defer procDeleteObject.Call(brushBg)
	brushCard, _, _ := procCreateSolidBrush.Call(uintptr(rgb(28, 34, 42)))
	defer procDeleteObject.Call(brushCard)
	brushThumb, _, _ := procCreateSolidBrush.Call(uintptr(rgb(48, 56, 68)))
	defer procDeleteObject.Call(brushThumb)
	brushBorder, _, _ := procCreateSolidBrush.Call(uintptr(rgb(100, 150, 110)))
	defer procDeleteObject.Call(brushBorder)
	brushBarBg, _, _ := procCreateSolidBrush.Call(uintptr(rgb(52, 62, 76)))
	defer procDeleteObject.Call(brushBarBg)
	brushPending, _, _ := procCreateSolidBrush.Call(uintptr(rgb(96, 118, 140)))
	defer procDeleteObject.Call(brushPending)
	brushWait, _, _ := procCreateSolidBrush.Call(uintptr(rgb(148, 132, 74)))
	defer procDeleteObject.Call(brushWait)
	brushDone, _, _ := procCreateSolidBrush.Call(uintptr(rgb(84, 138, 92)))
	defer procDeleteObject.Call(brushDone)
	brushFail, _, _ := procCreateSolidBrush.Call(uintptr(rgb(182, 82, 82)))
	defer procDeleteObject.Call(brushFail)
	brushSelected, _, _ := procCreateSolidBrush.Call(uintptr(rgb(54, 84, 66)))
	defer procDeleteObject.Call(brushSelected)
	brushSelectedBorder, _, _ := procCreateSolidBrush.Call(uintptr(rgb(128, 200, 138)))
	defer procDeleteObject.Call(brushSelectedBorder)

	procFillRect.Call(memDC, uintptr(unsafe.Pointer(&rc)), brushBg)
	procSetBkMode.Call(memDC, 1)

	visibleHeight := height
	scrollY := app.taskScrollSnapshot()

	filteredItems := app.filteredTaskItems()
	itemCount := len(filteredItems)
	rowHeight := taskCardHeight + taskCardGap
	if rowHeight <= 0 {
		rowHeight = 1
	}
	if itemCount > 0 {
		first := (scrollY - taskCardTopMargin) / rowHeight
		if first < 0 {
			first = 0
		}
		last := (scrollY + visibleHeight - taskCardTopMargin) / rowHeight
		if last < first {
			last = first
		}
		last += 2
		if last > itemCount {
			last = itemCount
		}
		items := filteredItems[first:last]
		for idx, item := range items {
			absoluteIdx := first + idx
			top := taskCardTopMargin + absoluteIdx*rowHeight - scrollY
			bottom := top + taskCardHeight
			if bottom < 0 || top > visibleHeight {
				continue
			}
			drawTaskCard(HDC(memDC), width, top, item, resolveTaskCardThumbnailPath(item), app.taskSelected(item.ID), brushCard, brushThumb, brushBorder, brushBarBg, brushPending, brushWait, brushDone, brushFail, brushSelected, brushSelectedBorder)
		}
	}

	procBitBlt.Call(
		paintDC,
		0, 0,
		uintptr(width),
		uintptr(height),
		memDC,
		0, 0,
		SRCCOPY,
	)
}

func drawTaskCard(hdc HDC, width, top int, item ui.TodoItem, thumbnailPath string, selected bool, brushCard, brushThumb, brushBorder, brushBarBg, brushPending, brushWait, brushDone, brushFail, brushSelected, brushSelectedBorder uintptr) {
	cardLeft := 10
	cardTop := top
	cardWidth := width - 20
	cardHeight := taskCardHeight
	if cardWidth < 100 {
		cardWidth = 100
	}
	cardRect := RECT{Left: int32(cardLeft), Top: int32(cardTop), Right: int32(cardLeft + cardWidth), Bottom: int32(cardTop + cardHeight)}
	if clipRegion, _, _ := procCreateRoundRectRgn.Call(
		uintptr(cardRect.Left+1),
		uintptr(cardRect.Top+1),
		uintptr(cardRect.Right-1),
		uintptr(cardRect.Bottom-1),
		uintptr(18),
		uintptr(18),
	); clipRegion != 0 {
		defer procDeleteObject.Call(clipRegion)
		procSelectClipRgn.Call(uintptr(hdc), clipRegion)
		defer procSelectClipRgn.Call(uintptr(hdc), 0)
	}
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&cardRect)), brushCard)
	if selected {
		selRect := RECT{Left: cardRect.Left + 1, Top: cardRect.Top + 1, Right: cardRect.Right - 1, Bottom: cardRect.Bottom - 1}
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&selRect)), brushSelected)
		leftBar := RECT{Left: cardRect.Left + 1, Top: cardRect.Top + 1, Right: cardRect.Left + 6, Bottom: cardRect.Bottom - 1}
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&leftBar)), brushSelectedBorder)
	}

	borderRect := RECT{Left: int32(cardLeft), Top: int32(cardTop), Right: int32(cardLeft + cardWidth), Bottom: int32(cardTop + 4)}
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&borderRect)), brushBorder)

	thumbRect := RECT{Left: int32(cardLeft + 10), Top: int32(cardTop + 12), Right: int32(cardLeft + 82), Bottom: int32(cardTop + 116)}
	drawTaskThumbnail(hdc, thumbRect, thumbnailPath, HBRUSH(brushThumb))

	contentRect := RECT{Left: int32(cardLeft + 90), Top: int32(cardTop + 12), Right: int32(cardLeft + cardWidth - 12), Bottom: int32(cardTop + cardHeight - 12)}
	contentFill, _, _ := procCreateSolidBrush.Call(uintptr(rgb(24, 30, 38)))
	if contentFill != 0 {
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&contentRect)), contentFill)
		procDeleteObject.Call(contentFill)
	}
	sepRect := RECT{Left: int32(cardLeft + 86), Top: int32(cardTop + 12), Right: int32(cardLeft + 88), Bottom: int32(cardTop + cardHeight - 12)}
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&sepRect)), brushBarBg)

	title := strings.TrimSpace(item.Result.Title)
	if title == "" {
		title = strings.TrimSpace(item.Request.URL)
	}
	if title == "" {
		title = item.ID
	}
	title = trimTextRunes(title, 56)
	sub := fmt.Sprintf("%s | %s", item.ID, strings.ToLower(string(item.Status)))
	siteLine := item.Request.URL
	if u, err := url.Parse(item.Request.URL); err == nil && u.Host != "" {
		siteLine = u.Host
	}
	siteLine = trimTextRunes(siteLine, 64)

	titleX := cardLeft + 96
	titleRight := cardLeft + cardWidth - 14
	drawTextLineWithFont(hdc, titleFont, titleX, cardTop+12, title, rgb(244, 247, 250))
	drawTextLineWithFont(hdc, uiFont, titleX, cardTop+36, siteLine, rgb(184, 196, 208))

	barY := cardTop + 58
	barRectBg := RECT{Left: int32(titleX), Top: int32(barY), Right: int32(titleRight), Bottom: int32(barY + 22)}
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&barRectBg)), brushBarBg)
	barRectFill := barRectBg
	progress := taskProgress(item)
	barRectFill.Right = barRectBg.Left + int32(float64(barRectBg.Right-barRectBg.Left)*progress)
	if barRectFill.Right < barRectFill.Left+2 {
		barRectFill.Right = barRectFill.Left + 2
	}
	switch item.Status {
	case ui.TodoStatusCompleted:
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&barRectFill)), brushDone)
	case ui.TodoStatusFailed:
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&barRectFill)), brushFail)
	case ui.TodoStatusPaused, ui.TodoStatusWaitingVerification, ui.TodoStatusVerificationCleared:
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&barRectFill)), brushWait)
	default:
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&barRectFill)), brushPending)
	}
	percent := fmt.Sprintf("%d%%", int(progress*100))
	drawTextLineWithFont(hdc, uiFont, cardLeft+96+(cardWidth-112)/2, cardTop+62, percent, rgb(252, 252, 252))
	statusText := taskProgressLabel(item)
	if statusText != "" {
		drawTextLineWithFont(hdc, uiFont, titleX, cardTop+86, sub, rgb(156, 168, 182))
		drawTextLineWithFont(hdc, uiFont, titleX, cardTop+106, statusText, rgb(184, 196, 208))
	} else {
		drawTextLineWithFont(hdc, uiFont, titleX, cardTop+86, sub, rgb(156, 168, 182))
	}
}

func resolveTaskCardThumbnailPath(item ui.TodoItem) string {
	if path := strings.TrimSpace(item.Result.ThumbnailPath); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if app == nil {
		return strings.TrimSpace(item.Result.ThumbnailPath)
	}
	current := app.paths.TaskThumbnailPath(item.ID)
	if _, err := os.Stat(current); err == nil {
		return current
	}
	legacy := filepath.Join(app.paths.Root, "runtime", "thumbnails", "task-"+item.ID, "thumb.jpg")
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	if path := strings.TrimSpace(item.Result.ThumbnailPath); path != "" {
		return path
	}
	return current
}

func taskProgress(item ui.TodoItem) float64 {
	if item.Progress > 0 {
		return clamp01(item.Progress)
	}
	switch item.Status {
	case ui.TodoStatusCompleted:
		return 1
	case ui.TodoStatusRunning:
		return 0.6
	case ui.TodoStatusFailed:
		return 1
	default:
		return 0.0
	}
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

func taskProgressLabel(item ui.TodoItem) string {
	switch {
	case item.Status == ui.TodoStatusFailed:
		return "失败"
	case item.Status == ui.TodoStatusCompleted && item.Result.DownloadedCount > 0:
		return fmt.Sprintf("%d张", item.Result.DownloadedCount)
	case item.StepTotal > 0 && item.StepCurrent > 0:
		base := fmt.Sprintf("%d/%d", item.StepCurrent, item.StepTotal)
		suffix := taskPhaseShort(item.Phase)
		if suffix == "" {
			return base
		}
		return base + " " + suffix
	case strings.TrimSpace(item.Phase) != "":
		return taskPhaseShort(item.Phase)
	default:
		return ""
	}
}

func taskPhaseShort(phase string) string {
	switch strings.TrimSpace(strings.ToLower(phase)) {
	case "pending":
		return "等待"
	case "queued":
		return "排队"
	case "routing":
		return "路由"
	case "preparing", "prep":
		return "准备"
	case "running":
		return "运行"
	case "completed", "done":
		return "完成"
	case "failed":
		return "失败"
	case "start":
		return "启动"
	case "parse":
		return "解析"
	case "downloading":
		return "下载中"
	case "active":
		return "激活"
	default:
		return phase
	}
}
func drawTextLine(hdc HDC, x, y int, text string, color uint32) {
	if text == "" {
		return
	}
	textPtr, _ := utf16Ptr(text)
	procSetTextColor.Call(uintptr(hdc), uintptr(color))
	procTextOutW.Call(uintptr(hdc), uintptr(x), uintptr(y), uintptr(unsafe.Pointer(textPtr)), uintptr(utf16Len(text)))
}

func drawTextLineWithFont(hdc HDC, font HGDIOBJ, x, y int, text string, color uint32) {
	if text == "" {
		return
	}
	textPtr, _ := utf16Ptr(text)
	oldFont, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(font))
	defer procSelectObject.Call(uintptr(hdc), oldFont)
	procSetTextColor.Call(uintptr(hdc), uintptr(color))
	procTextOutW.Call(uintptr(hdc), uintptr(x), uintptr(y), uintptr(unsafe.Pointer(textPtr)), uintptr(utf16Len(text)))
}

func hostTitle(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

func (a *frontendApp) filteredTaskItems() []ui.TodoItem {
	if a == nil || a.todo == nil {
		return nil
	}
	items := a.todo.Items()
	filter := a.taskSiteFilterSnapshot()
	if filter == "" {
		return items
	}
	filtered := make([]ui.TodoItem, 0, len(items))
	for _, item := range items {
		if taskSiteKey(item) == filter {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (a *frontendApp) taskSiteFilterSnapshot() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return strings.TrimSpace(strings.ToLower(a.taskSiteFilter))
}

func (a *frontendApp) setTaskSiteFilter(site string) {
	site = strings.TrimSpace(strings.ToLower(site))
	switch site {
	case "", "zeri", "nyahentai", "hentai2", "hentaiaz", "hitomi":
	default:
		site = ""
	}
	a.mu.Lock()
	a.taskSiteFilter = site
	a.selectedTaskIDs = map[string]bool{}
	a.taskScrollY = 0
	a.mu.Unlock()
	a.updateSiteFilterButtons()
	if a.taskBoard != 0 {
		procSetScrollPos.Call(uintptr(a.taskBoard), 1, 0, 1)
	}
	a.post(msgRefreshTasks)
	if site == "" {
		a.setStatus("showing all tasks")
		return
	}
	a.setStatus("showing %s tasks", site)
}

func (a *frontendApp) setupSiteFilterCombo() {
	if a.siteFilterCombo == 0 {
		return
	}
	for _, opt := range siteFilterOptions {
		text, _ := utf16Ptr(opt.Label)
		procSendMessageW.Call(uintptr(a.siteFilterCombo), CB_ADDSTRING, 0, uintptr(unsafe.Pointer(text)))
	}
	a.updateSiteFilterButtons()
}

func (a *frontendApp) updateSiteFilterButtons() {
	if a.siteFilterCombo == 0 {
		return
	}
	filter := a.taskSiteFilterSnapshot()
	index := 0
	for i, opt := range siteFilterOptions {
		if opt.Key == filter {
			index = i
			break
		}
	}
	procSendMessageW.Call(uintptr(a.siteFilterCombo), CB_SETCURSEL, uintptr(index), 0)
	procInvalidateRect.Call(uintptr(a.siteFilterCombo), 0, 1)
}

func (a *frontendApp) applySelectedSiteFilter() {
	if a.siteFilterCombo == 0 {
		return
	}
	r, _, _ := procSendMessageW.Call(uintptr(a.siteFilterCombo), CB_GETCURSEL, 0, 0)
	index := int(r)
	if index < 0 || index >= len(siteFilterOptions) {
		a.setTaskSiteFilter("")
		return
	}
	a.setTaskSiteFilter(siteFilterOptions[index].Key)
}

func taskSiteKey(item ui.TodoItem) string {
	for _, value := range []string{
		item.Result.Site,
		item.Request.URL,
		item.Result.URL,
		item.Result.ResolvedURL,
		item.Result.ReaderURL,
	} {
		if site := siteKeyFromText(value); site != "" {
			return site
		}
	}
	return ""
}

func siteKeyFromText(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	switch value {
	case "zeri", "nyahentai", "hentai2", "hentaiaz", "hitomi":
		return value
	}
	if u, err := url.Parse(value); err == nil && u.Host != "" {
		value = strings.ToLower(u.Host)
	}
	switch {
	case strings.Contains(value, "nyahentai"):
		return "nyahentai"
	case strings.Contains(value, "hentai2"):
		return "hentai2"
	case strings.Contains(value, "hentaiaz"):
		return "hentaiaz"
	case strings.Contains(value, "hitomi"):
		return "hitomi"
	case strings.Contains(value, "zeri"), strings.Contains(value, "zerobywzip"):
		return "zeri"
	default:
		return ""
	}
}

func trimTextRunes(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" || limit <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "..."
}

func taskBoardSnapshotScrollY() int {
	if app == nil {
		return 0
	}
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.taskScrollY
}

func (a *frontendApp) taskScrollSnapshot() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.taskScrollY
}

func rgb(r, g, b uint8) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func utf16Len(text string) int {
	if text == "" {
		return 0
	}
	return len(utf16.Encode([]rune(text)))
}

const (
	taskCardTopMargin = 10
	taskCardHeight    = 132
	taskCardGap       = 10
)
