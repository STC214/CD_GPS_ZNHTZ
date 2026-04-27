//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/tasks"
	"comic_downloader_go_playwright_stealth/ui"
)

const (
	windowClassName = "ComicDownloaderWin32Frontend"
	windowTitle     = "\u6f2b\u753b\u4e0b\u8f7d\u5668"

	menuIDSetExecutable     = 1001
	menuIDStartAll          = 1003
	menuIDSetDownloadDir    = 1004
	menuIDSetConcurrency    = 1005
	menuIDSetProgressDelay  = 1006
	menuIDRefreshAdblock    = 1007
	menuIDClearCompleted    = 1008
	menuIDInstallBrowsers   = 1009
	menuIDImportHistory     = 1010
	menuIDSetDriver         = 1011
	menuIDTaskRetry         = 1012
	menuIDTaskDetails       = 1013
	menuIDTaskOpenDir       = 1014
	menuIDTaskCopyURL       = 1015
	menuIDTaskDelete        = 1016
	menuIDTaskStart         = 1017
	menuIDTaskPause         = 1018
	menuIDSetPlaywrightRoot = 1019

	controlIDURLEdit        = 2001
	controlIDAddTask        = 2002
	controlIDTaskTitle      = 2003
	controlIDListBox        = 2004
	controlIDInfoTitle      = 2005
	controlIDInfoEdit       = 2006
	controlIDStatusText     = 2007
	controlIDStartTasks     = 2008
	controlIDDownloadDir    = 2010
	controlIDConcurrency    = 2011
	controlIDAdblockRules   = 2012
	controlIDClearDone      = 2013
	controlIDActionProgress = 2014
	controlIDSiteFilter     = 2015

	msgRefreshTasks  = 0x8001
	msgRefreshInfo   = 0x8002
	msgRefreshStatus = 0x8003
	msgRefreshBoard  = 0x8004
	msgRefreshAction = 0x8005
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")

	procAppendMenuW                             = user32.NewProc("AppendMenuW")
	procCreateMenu                              = user32.NewProc("CreateMenu")
	procCreatePopupMenu                         = user32.NewProc("CreatePopupMenu")
	procCreateWindowExW                         = user32.NewProc("CreateWindowExW")
	procBeginPaint                              = user32.NewProc("BeginPaint")
	procDefWindowProcW                          = user32.NewProc("DefWindowProcW")
	procDispatchMessageW                        = user32.NewProc("DispatchMessageW")
	procEndPaint                                = user32.NewProc("EndPaint")
	procGetClientRect                           = user32.NewProc("GetClientRect")
	procGetCursorPos                            = user32.NewProc("GetCursorPos")
	procGetKeyState                             = user32.NewProc("GetKeyState")
	procGetMessageW                             = user32.NewProc("GetMessageW")
	procGetModuleHandleW                        = kernel32.NewProc("GetModuleHandleW")
	procGetWindowPlacement                      = user32.NewProc("GetWindowPlacement")
	procGetWindowTextLen                        = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW                          = user32.NewProc("GetWindowTextW")
	procGetOpenFileNameW                        = comdlg32.NewProc("GetOpenFileNameW")
	procLoadCursorW                             = user32.NewProc("LoadCursorW")
	procLoadImageW                              = user32.NewProc("LoadImageW")
	procMessageBoxW                             = user32.NewProc("MessageBoxW")
	procMoveWindow                              = user32.NewProc("MoveWindow")
	procPostMessageW                            = user32.NewProc("PostMessageW")
	procPostQuitMessage                         = user32.NewProc("PostQuitMessage")
	procRegisterClassExW                        = user32.NewProc("RegisterClassExW")
	procSendMessageW                            = user32.NewProc("SendMessageW")
	procSetMenu                                 = user32.NewProc("SetMenu")
	procSetClassLongPtrW                        = user32.NewProc("SetClassLongPtrW")
	procSetWindowPlacement                      = user32.NewProc("SetWindowPlacement")
	procSetWindowTextW                          = user32.NewProc("SetWindowTextW")
	procShowWindow                              = user32.NewProc("ShowWindow")
	procTranslateMessage                        = user32.NewProc("TranslateMessage")
	procUpdateWindow                            = user32.NewProc("UpdateWindow")
	procInvalidateRect                          = user32.NewProc("InvalidateRect")
	procSetScrollRange                          = user32.NewProc("SetScrollRange")
	procSetScrollPos                            = user32.NewProc("SetScrollPos")
	procGetScrollPos                            = user32.NewProc("GetScrollPos")
	procTrackPopupMenu                          = user32.NewProc("TrackPopupMenu")
	procDestroyMenu                             = user32.NewProc("DestroyMenu")
	procSetWindowLongPtrW                       = user32.NewProc("SetWindowLongPtrW")
	procSHBrowseForFolderW                      = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW                    = shell32.NewProc("SHGetPathFromIDListW")
	procShellExecuteW                           = shell32.NewProc("ShellExecuteW")
	procCoTaskMemFree                           = ole32.NewProc("CoTaskMemFree")
	procSetCurrentProcessExplicitAppUserModelID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	gdi32                                       = syscall.NewLazyDLL("gdi32.dll")
	procCreateSolidBrush                        = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject                            = gdi32.NewProc("DeleteObject")
	procFillRect                                = user32.NewProc("FillRect")
	procSelectObject                            = gdi32.NewProc("SelectObject")
	procSetBkMode                               = gdi32.NewProc("SetBkMode")
	procSetBkColor                              = gdi32.NewProc("SetBkColor")
	procSetTextColor                            = gdi32.NewProc("SetTextColor")
	procTextOutW                                = gdi32.NewProc("TextOutW")
	procCreateFontW                             = gdi32.NewProc("CreateFontW")
	procCreateCompatibleDC                      = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap                  = gdi32.NewProc("CreateCompatibleBitmap")
	procBitBlt                                  = gdi32.NewProc("BitBlt")
	procDeleteDC                                = gdi32.NewProc("DeleteDC")
	procGetStockObject                          = gdi32.NewProc("GetStockObject")
	procDwmSetWindowAttribute                   = dwmapi.NewProc("DwmSetWindowAttribute")
	procSetPreferredAppMode                     = uxtheme.NewProc("SetPreferredAppMode")
	procFlushMenuThemes                         = uxtheme.NewProc("FlushMenuThemes")
	procAllowDarkModeForWindow                  = uxtheme.NewProc("AllowDarkModeForWindow")
)

var (
	gclpHIcon   = int32(-14)
	gclpHIconSm = int32(-34)
)

var (
	themeOnce      sync.Once
	themeInitErr   error
	darkBgBrush    HBRUSH
	darkPanelBrush HBRUSH
	darkEditBrush  HBRUSH
	activeBtnBrush HBRUSH
	uiFont         HGDIOBJ
	rowFont        HGDIOBJ
	titleFont      HGDIOBJ
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CLIPCHILDREN     = 0x02000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_TABSTOP          = 0x00010000
	WS_VSCROLL          = 0x00200000
	WS_EX_CLIENTEDGE    = 0x00000200
	WS_EX_DLGMODALFRAME = 0x00000001

	BS_PUSHBUTTON        = 0x00000000
	BS_AUTOCHECKBOX      = 0x00000003
	BS_FLAT              = 0x00008000
	CBS_DROPDOWNLIST     = 0x00000003
	CBS_HASSTRINGS       = 0x00000200
	LBS_NOTIFY           = 0x0001
	LBS_NOINTEGRALHEIGHT = 0x0100
	ES_AUTOHSCROLL       = 0x0080
	ES_READONLY          = 0x0800
	ES_MULTILINE         = 0x0004
	ES_AUTOVSCROLL       = 0x0040
	ES_WANTRETURN        = 0x1000

	WM_CREATE          = 0x0001
	WM_DESTROY         = 0x0002
	WM_CLOSE           = 0x0010
	WM_SIZE            = 0x0005
	WM_LBUTTONDOWN     = 0x0201
	WM_RBUTTONUP       = 0x0205
	WM_CONTEXTMENU     = 0x007B
	WM_MOUSEWHEEL      = 0x020A
	WM_GETMINMAXINFO   = 0x0024
	WM_COMMAND         = 0x0111
	WM_PAINT           = 0x000F
	WM_SETICON         = 0x0080
	WM_VSCROLL         = 0x0115
	WM_ERASEBKGND      = 0x0014
	WM_SETFONT         = 0x0030
	WM_CTLCOLORMSGBOX  = 0x0132
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLORDLG     = 0x0136
	WM_CTLCOLORSTATIC  = 0x0138
	EM_SETRECTNP       = 0x00B4
	EM_SETCUEBANNER    = 0x1501

	SW_SHOW = 5

	CW_USEDEFAULT = 0x80000000

	COLOR_WINDOW = 5
	IDC_ARROW    = 32512

	MF_STRING = 0x00000000
	MF_POPUP  = 0x00000010

	LB_RESETCONTENT = 0x0184
	LB_ADDSTRING    = 0x0180
	CB_GETCURSEL    = 0x0147
	CB_ADDSTRING    = 0x0143
	CB_SETCURSEL    = 0x014E
	CBN_SELCHANGE   = 1

	SB_LINEUP        = 0
	SB_LINEDOWN      = 1
	SB_PAGEUP        = 2
	SB_PAGEDOWN      = 3
	SB_THUMBPOSITION = 4
	SB_THUMBTRACK    = 5
	SB_TOP           = 6
	SB_BOTTOM        = 7
	SB_ENDSCROLL     = 8

	MB_OK              = 0x00000000
	MB_YESNO           = 0x00000004
	MB_ICONQUESTION    = 0x00000020
	MB_ICONINFORMATION = 0x00000040

	ICON_SMALL = 0
	ICON_BIG   = 1

	IMAGE_ICON      = 1
	LR_LOADFROMFILE = 0x00000010
	LR_DEFAULTSIZE  = 0x00000040
	LR_SHARED       = 0x00008000
	SRCCOPY         = 0x00CC0020
	TPM_LEFTALIGN   = 0x0000
	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100
	TPM_NONOTIFY    = 0x0080

	DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	DWMWA_CAPTION_COLOR           = 35
	DWMWA_TEXT_COLOR              = 36

	APPMODE_DEFAULT    = 0
	APPMODE_ALLOWDARK  = 1
	APPMODE_FORCEDARK  = 2
	APPMODE_FORCELIGHT = 3
	APPMODE_MAX        = 4

	OFN_EXPLORER      = 0x00080000
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_PATHMUSTEXIST = 0x00000800

	BIF_RETURNONLYFSDIRS = 0x0001
	BIF_USENEWUI         = 0x0040

	VK_CONTROL     = 0x11
	VK_ESCAPE      = 0x1B
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	SW_SHOWNORMAL  = 1
)

type HWND uintptr
type HINSTANCE uintptr
type HCURSOR uintptr
type HBRUSH uintptr
type HDC uintptr
type HGDIOBJ uintptr

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type POINT struct {
	X int32
	Y int32
}

type MINMAXINFO struct {
	PtReserved     POINT
	PtMaxSize      POINT
	PtMaxPosition  POINT
	PtMinTrackSize POINT
	PtMaxTrackSize POINT
}

type WINDOWPLACEMENT struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    POINT
	PtMaxPosition    POINT
	RcNormalPosition RECT
}

type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         uintptr
	HCursor       HCURSOR
	HbrBackground HBRUSH
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type PAINTSTRUCT struct {
	Hdc         HDC
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type OPENFILENAMEW struct {
	LStructSize       uint32
	HwndOwner         HWND
	HInstance         HINSTANCE
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

type BROWSEINFOW struct {
	HwndOwner      HWND
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type CREATESTRUCT struct {
	CreateParams uintptr
	Instance     HINSTANCE
	Menu         uintptr
	Parent       HWND
	Cy           int32
	Cx           int32
	Y            int32
	X            int32
	Style        int32
	LpszName     *uint16
	LpszClass    *uint16
	ExStyle      uint32
}

type frontendApp struct {
	workspaceRoot string
	paths         projectruntime.Paths
	menuState     ui.BrowserMenuState
	installMW     ui.BrowserInstallMiddleware
	todo          *ui.TodoList
	downloadDir   string
	concurrency   int
	progressDelay time.Duration
	adblockInfo   string

	hwnd HWND

	urlEdit         HWND
	downloadDirBtn  HWND
	concurrencyBtn  HWND
	adblockBtn      HWND
	clearDoneBtn    HWND
	addTaskBtn      HWND
	startTasksBtn   HWND
	refreshBtn      HWND
	actionProgress  HWND
	taskTitle       HWND
	siteFilterCombo HWND
	taskBoard       HWND
	infoTitle       HWND
	infoEdit        HWND
	statusText      HWND

	taskScrollY     int
	taskContentH    int
	selectedTaskIDs map[string]bool
	taskSiteFilter  string

	actionProgressValue float64
	actionProgressText  string
	frontendStatePath   string
	windowPlacement     projectruntime.FrontendWindowPlacement
	windowPlacementSet  bool
	lastInfoText        string

	mu     sync.RWMutex
	status string
	hIcon  uintptr
}

var app *frontendApp

var wndProc = syscall.NewCallback(windowProc)

func main() {
	stdruntime.LockOSThread()
	workspaceRoot, err := executableWorkspaceRoot()
	if err != nil {
		log.Fatalf("resolve workspace root: %v", err)
	}
	if cleanup, logPath, err := projectruntime.InitProcessLogging(workspaceRoot, "win32-frontend"); err != nil {
		log.Fatalf("init frontend logging: %v", err)
	} else {
		log.Printf("frontend logging: %s", logPath)
		defer func() {
			if err := cleanup(); err != nil {
				log.Printf("close frontend log: %v", err)
			}
		}()
	}
	app = newFrontendApp(workspaceRoot)
	app.todo.SetNotifier(func() {
		app.post(msgRefreshTasks)
	})
	if err := app.run(); err != nil {
		log.Fatal(err)
	}
}

func newFrontendApp(workspaceRoot string) *frontendApp {
	paths := projectruntime.NewPaths(workspaceRoot)
	_ = paths.Ensure()
	menu := ui.DefaultBrowserMenuState().
		WithFirefoxExecutablePath(projectruntime.DefaultFirefoxExecutablePath(paths.Root)).
		WithFirefoxInstallRoot(projectruntime.DefaultFirefoxInstallDir(paths.Root))
	app := &frontendApp{
		workspaceRoot:     workspaceRoot,
		paths:             paths,
		menuState:         menu,
		installMW:         ui.NewBrowserInstallMiddleware(workspaceRoot),
		todo:              ui.NewTodoList(),
		downloadDir:       projectruntime.DefaultDownloadDir(workspaceRoot),
		concurrency:       1,
		progressDelay:     80 * time.Millisecond,
		adblockInfo:       "rules not loaded",
		selectedTaskIDs:   map[string]bool{},
		status:            "ready",
		frontendStatePath: projectruntime.ResolveFrontendStatePath(workspaceRoot),
	}
	app.todo.SetRuntimeRoot(projectruntime.ResolveRuntimeRoot(workspaceRoot))
	app.todo.SetConcurrencyLimit(app.concurrency)
	app.todo.SetProgressNotifyDelay(app.progressDelay)
	if state, err := projectruntime.LoadFrontendState(app.frontendStatePath); err == nil {
		app.applyFrontendState(state)
	}
	legacyPath := projectruntime.ResolveLegacyComicDownloaderStatePath(workspaceRoot)
	if count, err := app.todo.LoadLegacyComicDownloaderHistory(legacyPath); err != nil {
		log.Printf("load legacy task history %q failed: %v", legacyPath, err)
	} else if count > 0 {
		app.setStatus("loaded %d legacy tasks from %s", count, legacyPath)
		if err := app.todo.SaveLegacyComicDownloaderState(legacyPath, app.concurrency); err != nil {
			log.Printf("rewrite legacy task history %q failed: %v", legacyPath, err)
		} else {
			log.Printf("rewrote legacy task history with normalized URLs: %s", legacyPath)
		}
	}
	return app
}

func (a *frontendApp) applyFrontendState(state projectruntime.FrontendState) {
	a.mu.Lock()
	if strings.TrimSpace(state.SelectedBrowser) != "" {
		a.menuState = a.menuState.WithSelectedBrowser(state.SelectedBrowser)
	}
	if strings.TrimSpace(state.FirefoxExecutablePath) != "" {
		a.menuState = a.menuState.WithFirefoxExecutablePath(state.FirefoxExecutablePath)
	}
	if strings.TrimSpace(state.FirefoxInstallRoot) != "" {
		a.menuState = a.menuState.WithFirefoxInstallRoot(state.FirefoxInstallRoot)
	}
	if strings.TrimSpace(state.PlaywrightDriverDir) != "" {
		a.menuState = a.menuState.WithPlaywrightDriverDir(state.PlaywrightDriverDir)
	}
	a.menuState = normalizeBrowserMenuState(a.menuState)
	if strings.TrimSpace(state.DownloadDir) != "" {
		a.downloadDir = projectruntime.ResolvePath(a.workspaceRoot, state.DownloadDir)
	}
	if state.Concurrency > 0 {
		a.concurrency = state.Concurrency
	}
	if state.ProgressRefreshMs > 0 {
		a.progressDelay = time.Duration(state.ProgressRefreshMs) * time.Millisecond
	}
	a.windowPlacement = state.WindowPlacement
	a.windowPlacementSet = !frontendWindowPlacementZero(state.WindowPlacement)
	a.mu.Unlock()
	a.todo.SetConcurrencyLimit(a.currentConcurrency())
	a.todo.SetProgressNotifyDelay(a.currentProgressDelay())
}

func (a *frontendApp) persistFrontendState() {
	if strings.TrimSpace(a.frontendStatePath) == "" {
		return
	}
	a.mu.RLock()
	menu := a.menuState
	downloadDir := a.downloadDir
	concurrency := a.concurrency
	progressRefreshMs := int(a.progressDelay / time.Millisecond)
	a.mu.RUnlock()
	state := projectruntime.FrontendState{
		Version:               1,
		SelectedBrowser:       menu.SelectedBrowser,
		FirefoxExecutablePath: menu.FirefoxExecutablePath,
		FirefoxInstallRoot:    menu.FirefoxInstallRoot,
		PlaywrightDriverDir:   menu.PlaywrightDriverDir,
		DownloadDir:           projectruntime.RelativizePath(a.workspaceRoot, downloadDir),
		Concurrency:           concurrency,
		ProgressRefreshMs:     progressRefreshMs,
		WindowPlacement:       a.currentWindowPlacement(),
	}
	if err := projectruntime.SaveFrontendState(a.frontendStatePath, state); err != nil {
		log.Printf("save frontend state failed: %v", err)
	}
}

func (a *frontendApp) persistTaskHistory() {
	if a.todo == nil {
		return
	}
	path := projectruntime.ResolveLegacyComicDownloaderStatePath(a.workspaceRoot)
	a.mu.RLock()
	concurrency := a.concurrency
	a.mu.RUnlock()
	if err := a.todo.SaveLegacyComicDownloaderState(path, concurrency); err != nil {
		log.Printf("save task history failed: %v", err)
	}
}

func (a *frontendApp) currentWindowPlacement() projectruntime.FrontendWindowPlacement {
	if a.hwnd == 0 {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.windowPlacement
	}
	placement, ok := getWindowPlacementSnapshot(a.hwnd)
	if !ok {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.windowPlacement
	}
	a.mu.Lock()
	a.windowPlacement = placement
	a.windowPlacementSet = !frontendWindowPlacementZero(placement)
	a.mu.Unlock()
	return placement
}

func (a *frontendApp) restoreWindowPlacement() {
	if a.hwnd == 0 {
		return
	}
	a.mu.RLock()
	placement := a.windowPlacement
	ok := a.windowPlacementSet
	a.mu.RUnlock()
	if !ok || frontendWindowPlacementZero(placement) {
		return
	}
	_ = setWindowPlacement(a.hwnd, placement)
}

func frontendWindowPlacementZero(p projectruntime.FrontendWindowPlacement) bool {
	return p.Flags == 0 && p.ShowCmd == 0 &&
		p.MinPositionX == 0 && p.MinPositionY == 0 &&
		p.MaxPositionX == 0 && p.MaxPositionY == 0 &&
		p.NormalPositionLeft == 0 && p.NormalPositionTop == 0 &&
		p.NormalPositionRight == 0 && p.NormalPositionBottom == 0
}

func (a *frontendApp) run() error {
	if err := ensureThemeResources(); err != nil {
		return err
	}
	defer shutdownGDIPlus()
	defer shutdownTaskThumbnailCache()
	enableDarkAppMode()
	_ = setExplicitAppUserModelID("ComicDownloader.Portable")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	iconPath := filepath.Join(a.paths.Root, "app.ico")
	iconHandle, iconErr := loadAppIcon(iconPath)
	if iconErr != nil {
		log.Printf("app icon unavailable: %v", iconErr)
	}
	a.hIcon = iconHandle
	if err := registerTaskBoardClass(HINSTANCE(hInstance)); err != nil {
		return err
	}
	if err := registerActionProgressClass(HINSTANCE(hInstance)); err != nil {
		return err
	}
	className, err := utf16Ptr(windowClassName)
	if err != nil {
		return err
	}
	title, err := utf16Ptr(windowTitle)
	if err != nil {
		return err
	}
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   wndProc,
		HInstance:     HINSTANCE(hInstance),
		HIcon:         a.hIcon,
		HCursor:       HCURSOR(cursor),
		HbrBackground: darkBgBrush,
		LpszClassName: className,
		HIconSm:       a.hIcon,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW: %w", err)
	}
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE|WS_CLIPCHILDREN,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		450,
		900,
		0,
		0,
		hInstance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW: %w", err)
	}
	a.hwnd = HWND(hwnd)
	if a.hIcon != 0 {
		procSetClassLongPtrW.Call(hwnd, uintptr(gclpHIcon), a.hIcon)
		procSetClassLongPtrW.Call(hwnd, uintptr(gclpHIconSm), a.hIcon)
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, a.hIcon)
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, a.hIcon)
	}
	applyDarkWindowChrome(a.hwnd)
	a.restoreWindowPlacement()
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	var msg MSG
	for {
		r, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(r) {
		case -1:
			return fmt.Errorf("GetMessageW: %w", err)
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func windowProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		app.hwnd = hwnd
		app.attachMenu(hwnd)
		app.createControls(hwnd)
		app.refreshAllUI()
		return 0
	case WM_ERASEBKGND:
		if darkBgBrush != 0 {
			var rc RECT
			if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r != 0 {
				procFillRect.Call(wParam, uintptr(unsafe.Pointer(&rc)), uintptr(darkBgBrush))
			}
		}
		return 1
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN:
		procSetTextColor.Call(wParam, uintptr(rgb(238, 242, 247)))
		procSetBkMode.Call(wParam, 1)
		procSetBkColor.Call(wParam, uintptr(rgb(18, 22, 28)))
		if msg == WM_CTLCOLORBTN {
			return uintptr(darkPanelBrush)
		}
		return uintptr(darkBgBrush)
	case WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		procSetTextColor.Call(wParam, uintptr(rgb(245, 247, 250)))
		procSetBkMode.Call(wParam, 2)
		procSetBkColor.Call(wParam, uintptr(rgb(36, 42, 52)))
		return uintptr(darkEditBrush)
	case WM_SIZE:
		app.layout()
		return 0
	case WM_GETMINMAXINFO:
		if lParam != 0 {
			mmi := (*MINMAXINFO)(unsafe.Pointer(lParam))
			mmi.PtMinTrackSize.X = 450
			mmi.PtMinTrackSize.Y = 620
		}
		return 0
	case WM_COMMAND:
		app.handleCommand(uint16(wParam&0xFFFF), uint16((wParam>>16)&0xFFFF))
		return 0
	case msgRefreshTasks:
		app.refreshTaskList()
		return 0
	case msgRefreshInfo:
		app.refreshInfo()
		return 0
	case msgRefreshStatus:
		app.refreshStatus()
		return 0
	case msgRefreshAction:
		app.refreshActionProgress()
		return 0
	case WM_DESTROY:
		app.persistTaskHistory()
		app.persistFrontendState()
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func (a *frontendApp) attachMenu(hwnd HWND) {
	mainMenu, _, _ := procCreateMenu.Call()
	browserMenu, _, _ := procCreatePopupMenu.Call()
	settingsMenu, _, _ := procCreatePopupMenu.Call()
	taskMenu, _, _ := procCreatePopupMenu.Call()

	addMenuItem(browserMenu, menuIDSetExecutable, "\u8bbe\u7f6e Firefox \u53ef\u6267\u884c\u6587\u4ef6...")
	addMenuItem(browserMenu, menuIDSetPlaywrightRoot, "\u8bbe\u7f6e Playwright \u6839\u76ee\u5f55...")
	addMenuItem(browserMenu, menuIDInstallBrowsers, "\u5b89\u88c5\u6d4f\u89c8\u5668...")
	addMenuItem(browserMenu, menuIDSetDriver, "\u8bbe\u7f6e Playwright driver \u76ee\u5f55...")
	addMenuItem(settingsMenu, menuIDSetDownloadDir, "\u8bbe\u7f6e\u4e0b\u8f7d\u76ee\u5f55...")
	addMenuItem(settingsMenu, menuIDSetConcurrency, "\u8bbe\u7f6e\u5e76\u53d1\u6570...")
	addMenuItem(settingsMenu, menuIDSetProgressDelay, "\u8bbe\u7f6e\u8fdb\u5ea6\u5237\u65b0\u95f4\u9694...")
	addMenuItem(settingsMenu, menuIDRefreshAdblock, "\u66f4\u65b0\u5e7f\u544a\u62e6\u622a\u89c4\u5219")
	addMenuItem(taskMenu, menuIDStartAll, "\u5f00\u59cb\u6240\u6709\u672a\u5b8c\u6210\u4efb\u52a1")
	addMenuItem(taskMenu, menuIDImportHistory, "\u5bfc\u5165\u5386\u53f2\u8bb0\u5f55...")
	addMenuItem(taskMenu, menuIDClearCompleted, "\u6e05\u7406\u5df2\u5b8c\u6210\u4efb\u52a1")

	browserLabel, _ := utf16Ptr("\u6d4f\u89c8\u5668")
	settingsLabel, _ := utf16Ptr("\u8bbe\u7f6e")
	taskLabel, _ := utf16Ptr("\u4efb\u52a1")
	procAppendMenuW.Call(mainMenu, MF_POPUP, browserMenu, uintptr(unsafe.Pointer(browserLabel)))
	procAppendMenuW.Call(mainMenu, MF_POPUP, settingsMenu, uintptr(unsafe.Pointer(settingsLabel)))
	procAppendMenuW.Call(mainMenu, MF_POPUP, taskMenu, uintptr(unsafe.Pointer(taskLabel)))
	procSetMenu.Call(uintptr(hwnd), mainMenu)
}

func addMenuItem(menu uintptr, id uintptr, title string) {
	text, _ := utf16Ptr(title)
	procAppendMenuW.Call(menu, MF_STRING, id, uintptr(unsafe.Pointer(text)))
}

func (a *frontendApp) createControls(hwnd HWND) {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	def := []struct {
		class   string
		text    string
		style   uintptr
		exStyle uintptr
		x, y    int32
		w, h    int32
		id      int
	}{
		{class: "Edit", text: "", style: WS_CHILD | WS_VISIBLE | WS_BORDER | ES_AUTOHSCROLL | ES_MULTILINE | ES_AUTOVSCROLL | ES_WANTRETURN, exStyle: WS_EX_CLIENTEDGE, x: 20, y: 8, w: 980, h: 25, id: controlIDURLEdit},
		{class: "Button", text: "\u6dfb\u52a0\u4efb\u52a1", style: WS_CHILD | WS_VISIBLE | BS_PUSHBUTTON | WS_TABSTOP, x: 900, y: 8, w: 120, h: 25, id: controlIDAddTask},
		{class: "Static", text: "\u4efb\u52a1\u5217\u8868", style: WS_CHILD | WS_VISIBLE, x: 20, y: 70, w: 120, h: 20, id: controlIDTaskTitle},
		{class: "ComboBox", text: "", style: WS_CHILD | WS_VISIBLE | WS_VSCROLL | CBS_DROPDOWNLIST | CBS_HASSTRINGS | WS_TABSTOP, x: 122, y: 48, w: 116, h: 150, id: controlIDSiteFilter},
		{class: "TaskBoardControl", text: "", style: WS_CHILD | WS_VISIBLE | WS_BORDER | WS_VSCROLL, exStyle: WS_EX_CLIENTEDGE, x: 20, y: 88, w: 1240, h: 300, id: controlIDListBox},
		{class: "Static", text: "ready", style: WS_CHILD | WS_VISIBLE, x: 20, y: 862, w: 1240, h: 24, id: controlIDStatusText},
	}

	for _, c := range def {
		className, _ := utf16Ptr(c.class)
		text, _ := utf16Ptr(c.text)
		hwndChild, _, err := procCreateWindowExW.Call(
			c.exStyle,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(text)),
			c.style,
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
			log.Printf("create %s failed: %v", c.class, err)
			continue
		}
		switch c.id {
		case controlIDURLEdit:
			a.urlEdit = HWND(hwndChild)
		case controlIDAddTask:
			a.addTaskBtn = HWND(hwndChild)
		case controlIDTaskTitle:
			a.taskTitle = HWND(hwndChild)
		case controlIDSiteFilter:
			a.siteFilterCombo = HWND(hwndChild)
		case controlIDListBox:
			a.taskBoard = HWND(hwndChild)
		case controlIDStatusText:
			a.statusText = HWND(hwndChild)
		}
		if c.id == controlIDURLEdit || c.id == controlIDAddTask || c.id == controlIDSiteFilter {
			setControlFont(HWND(hwndChild), rowFont)
		} else {
			setControlFont(HWND(hwndChild), uiFont)
		}
	}
	a.setupSiteFilterCombo()
	setControlFont(a.taskTitle, uiFont)
	setControlFont(a.taskBoard, uiFont)
	setControlFont(a.statusText, uiFont)
	a.updateConcurrencyButton(a.concurrency)
	a.updateSiteFilterButtons()
}

func (a *frontendApp) layout() {
	var rc RECT
	if r, _, _ := procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return
	}
	width := int32(rc.Right - rc.Left)
	height := int32(rc.Bottom - rc.Top)

	padding := int32(20)
	addW := int32(78)
	gap := int32(12)
	buttonX := width - padding - addW
	if buttonX < 20 {
		buttonX = 20
	}
	urlW := buttonX - 20 - gap
	if urlW < 180 {
		urlW = 180
	}
	move(a.urlEdit, 20, 8, urlW, 25)
	move(a.addTaskBtn, buttonX, 8, addW, 25)
	adjustURLInputTextRect(a.urlEdit, 8, 4, 8, 3)
	setCueBanner(a.urlEdit, "输入漫画页面 URL")
	taskTop := int32(74)
	statusY := height - 34
	taskBottom := statusY - 10
	taskH := max32(160, taskBottom-taskTop)
	titleW := int32(60)
	move(a.taskTitle, padding, 50, titleW, 20)
	filterW := int32(116)
	filterX := padding + titleW + 42
	if maxX := width - 8 - filterW; filterX > maxX {
		filterX = maxX
	}
	if minX := padding + titleW + 8; filterX < minX {
		filterX = minX
	}
	move(a.siteFilterCombo, filterX, 48, filterW, 150)
	move(a.taskBoard, padding, taskTop, width-padding*2, taskH)
	move(a.statusText, padding, statusY, width-padding*2, 24)
	taskBoardRefresh(a.taskBoard)
}

func move(hwnd HWND, x, y, w, h int32) {
	if hwnd == 0 {
		return
	}
	procMoveWindow.Call(uintptr(hwnd), uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
}

func getWindowPlacementSnapshot(hwnd HWND) (projectruntime.FrontendWindowPlacement, bool) {
	if hwnd == 0 {
		return projectruntime.FrontendWindowPlacement{}, false
	}
	wp := WINDOWPLACEMENT{Length: uint32(unsafe.Sizeof(WINDOWPLACEMENT{}))}
	r, _, _ := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wp)))
	if r == 0 {
		return projectruntime.FrontendWindowPlacement{}, false
	}
	return projectruntime.FrontendWindowPlacement{
		Flags:                wp.Flags,
		ShowCmd:              wp.ShowCmd,
		MinPositionX:         wp.PtMinPosition.X,
		MinPositionY:         wp.PtMinPosition.Y,
		MaxPositionX:         wp.PtMaxPosition.X,
		MaxPositionY:         wp.PtMaxPosition.Y,
		NormalPositionLeft:   wp.RcNormalPosition.Left,
		NormalPositionTop:    wp.RcNormalPosition.Top,
		NormalPositionRight:  wp.RcNormalPosition.Right,
		NormalPositionBottom: wp.RcNormalPosition.Bottom,
	}, true
}

func setWindowPlacement(hwnd HWND, placement projectruntime.FrontendWindowPlacement) error {
	if hwnd == 0 {
		return fmt.Errorf("window handle is empty")
	}
	wp := WINDOWPLACEMENT{
		Length:  uint32(unsafe.Sizeof(WINDOWPLACEMENT{})),
		Flags:   placement.Flags,
		ShowCmd: placement.ShowCmd,
		PtMinPosition: POINT{
			X: placement.MinPositionX,
			Y: placement.MinPositionY,
		},
		PtMaxPosition: POINT{
			X: placement.MaxPositionX,
			Y: placement.MaxPositionY,
		},
		RcNormalPosition: RECT{
			Left:   placement.NormalPositionLeft,
			Top:    placement.NormalPositionTop,
			Right:  placement.NormalPositionRight,
			Bottom: placement.NormalPositionBottom,
		},
	}
	r, _, err := procSetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wp)))
	if r == 0 {
		return fmt.Errorf("SetWindowPlacement: %w", err)
	}
	return nil
}

func adjustURLInputTextRect(hwnd HWND, left, top, right, bottom int32) {
	if hwnd == 0 {
		return
	}
	var rc RECT
	if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return
	}
	rc.Left = left
	rc.Top = top
	rc.Right = max32(left+20, rc.Right-right)
	rc.Bottom = max32(top+10, rc.Bottom-bottom)
	procSendMessageW.Call(uintptr(hwnd), uintptr(EM_SETRECTNP), 0, uintptr(unsafe.Pointer(&rc)))
}

func setCueBanner(hwnd HWND, text string) {
	if hwnd == 0 {
		return
	}
	ptr, err := utf16Ptr(text)
	if err != nil {
		return
	}
	procSendMessageW.Call(uintptr(hwnd), uintptr(EM_SETCUEBANNER), 1, uintptr(unsafe.Pointer(ptr)))
}

func (a *frontendApp) handleCommand(id uint16, notifyCode uint16) {
	switch id {
	case menuIDSetExecutable:
		a.pickFirefoxExecutable()
	case menuIDSetPlaywrightRoot:
		a.pickPlaywrightRoot()
	case menuIDInstallBrowsers:
		a.installBrowsersAsync()
	case menuIDSetDriver:
		a.pickPlaywrightDriverDir()
	case menuIDStartAll:
		a.startPendingAsync()
	case menuIDImportHistory:
		a.importLegacyHistoryAsync()
	case menuIDSetDownloadDir:
		a.pickDownloadDir()
	case menuIDSetConcurrency:
		a.promptConcurrency()
	case menuIDSetProgressDelay:
		a.promptProgressDelay()
	case menuIDRefreshAdblock:
		a.refreshAdblockRules()
	case menuIDClearCompleted:
		a.clearCompletedTasks()
	case controlIDAddTask:
		a.addPendingTask()
	case controlIDStartTasks:
		a.startPendingAsync()
	case controlIDDownloadDir:
		a.pickDownloadDir()
	case controlIDConcurrency:
		a.promptConcurrency()
	case controlIDAdblockRules:
		a.refreshAdblockRules()
	case controlIDClearDone:
		a.clearCompletedTasks()
	case controlIDSiteFilter:
		if notifyCode == CBN_SELCHANGE {
			a.applySelectedSiteFilter()
		}
	}
}

func (a *frontendApp) showValue(title, value string) {
	msgBox(a.hwnd, title+"\r\n\r\n"+value, title)
}

func (a *frontendApp) pickFirefoxExecutable() {
	path, err := openFileDialog(a.hwnd, "Select Firefox executable", "Executable Files (*.exe)\x00*.exe\x00All Files (*.*)\x00*.*\x00\x00", a.menuState.FirefoxExecutablePath)
	if err != nil {
		a.setStatus("select Firefox executable failed: %v", err)
		return
	}
	if strings.TrimSpace(path) == "" {
		a.setStatus("Firefox executable unchanged")
		return
	}
	a.mu.Lock()
	a.menuState = a.menuState.WithFirefoxExecutablePath(path)
	a.mu.Unlock()
	a.setStatus("Firefox executable set: %s", path)
	a.refreshUnfinishedTaskBrowserSettings()
	a.persistFrontendState()
	a.post(msgRefreshInfo)
}

func (a *frontendApp) pickPlaywrightRoot() {
	path, err := browseFolderDialog(a.hwnd, "Select Playwright browser root", a.defaultPlaywrightRoot())
	if err != nil {
		a.setStatus("select Playwright root failed: %v", err)
		return
	}
	if strings.TrimSpace(path) == "" {
		a.setStatus("Playwright root unchanged")
		return
	}
	a.mu.RLock()
	state := a.menuState
	a.mu.RUnlock()
	next, resolved, err := state.WithPlaywrightRoot(path)
	if err != nil {
		a.setStatus("resolve Playwright root failed: %v", err)
		msgBox(a.hwnd, fmt.Sprintf("Cannot find Firefox and Playwright driver under:\r\n\r\n%s\r\n\r\n%v", path, err), "Playwright Root")
		return
	}
	a.mu.Lock()
	a.menuState = next
	a.mu.Unlock()
	a.setStatus("Playwright root set: firefox=%s driver=%s", resolved.FirefoxExecutable, resolved.PlaywrightDriverDir)
	a.refreshUnfinishedTaskBrowserSettings()
	a.persistFrontendState()
	a.post(msgRefreshInfo)
	msgBox(a.hwnd, fmt.Sprintf("Playwright root configured.\r\n\r\nFirefox:\r\n%s\r\n\r\nDriver:\r\n%s", resolved.FirefoxExecutable, resolved.PlaywrightDriverDir), "Playwright Root")
}

func (a *frontendApp) pickPlaywrightDriverDir() {
	path, err := browseFolderDialog(a.hwnd, "Select Playwright driver directory", a.menuState.PlaywrightDriverDir)
	if err != nil {
		a.setStatus("select Playwright driver directory failed: %v", err)
		return
	}
	if strings.TrimSpace(path) == "" {
		a.setStatus("Playwright driver directory unchanged")
		return
	}
	a.mu.RLock()
	state := a.menuState
	a.mu.RUnlock()
	if next, resolved, err := state.WithPlaywrightRoot(path); err == nil {
		a.mu.Lock()
		a.menuState = next
		a.mu.Unlock()
		a.setStatus("Playwright root set from driver picker: firefox=%s driver=%s", resolved.FirefoxExecutable, resolved.PlaywrightDriverDir)
		a.refreshUnfinishedTaskBrowserSettings()
		a.persistFrontendState()
		a.post(msgRefreshInfo)
		return
	}
	driverDir, err := ui.ResolvePlaywrightDriverDir(path)
	if err != nil {
		a.setStatus("resolve Playwright driver directory failed: %v", err)
		msgBox(a.hwnd, fmt.Sprintf("Cannot find Playwright driver under:\r\n\r\n%s\r\n\r\n%v", path, err), "Playwright Driver")
		return
	}
	a.mu.Lock()
	a.menuState = a.menuState.WithPlaywrightDriverDir(driverDir)
	a.mu.Unlock()
	a.setStatus("Playwright driver directory set: %s", driverDir)
	a.refreshUnfinishedTaskBrowserSettings()
	a.persistFrontendState()
	a.post(msgRefreshInfo)
}

func (a *frontendApp) installBrowsersAsync() {
	go func() {
		path, err := browseFolderDialog(a.hwnd, "Select browser install directory", a.defaultBrowserInstallRoot())
		if err != nil {
			a.setStatus("select browser install directory failed: %v", err)
			return
		}
		if strings.TrimSpace(path) == "" {
			a.setStatus("browser install directory unchanged")
			return
		}
		a.setActionProgress(0.05, "选择目录")
		a.setStatus("installing browsers into %s ...", path)
		a.mu.RLock()
		stateSnapshot := a.menuState
		a.mu.RUnlock()
		newState, result, err := a.installMW.InstallAllBrowsersAndApply(stateSnapshot, path, func(update projectruntime.BrowserInstallProgress) {
			phase := strings.TrimSpace(update.Phase)
			message := strings.TrimSpace(update.Message)
			label := "安装浏览器"
			if update.Browser != "" {
				label = browserTypeLabel(update.Browser)
			}
			if phase != "" {
				if message != "" {
					a.setActionProgress(update.Fraction, label+" "+phase)
					a.setStatus("%s: %s", label, message)
					return
				}
				a.setActionProgress(update.Fraction, label+" "+phase)
				a.setStatus("%s: %s", label, phase)
				return
			}
			if message != "" {
				a.setActionProgress(update.Fraction, label)
				a.setStatus("%s: %s", label, message)
				return
			}
			a.setActionProgress(update.Fraction, label)
		})
		if err != nil {
			a.setActionProgress(1, "失败")
			a.setStatus("install browsers failed: %v", err)
			return
		}
		a.mu.Lock()
		a.menuState = newState
		a.mu.Unlock()
		a.refreshUnfinishedTaskBrowserSettings()
		a.setActionProgress(1, "完成")
		a.persistFrontendState()
		a.setStatus("installed browsers into %s", result.TargetRoot)
		msgBox(a.hwnd, "Browsers installed successfully\r\n\r\n"+result.TargetRoot, "Success")
		a.post(msgRefreshInfo)
	}()
}

func (a *frontendApp) refreshUnfinishedTaskBrowserSettings() int {
	if a == nil || a.todo == nil {
		return 0
	}
	a.mu.RLock()
	menu := a.menuState
	downloadDir := a.downloadDir
	runtimeRoot := a.paths.Root
	a.mu.RUnlock()
	changed := a.todo.RefreshUnfinishedRequests(func(req tasks.BrowserLaunchRequest) tasks.BrowserLaunchRequest {
		req.BrowserType = browserTypeForTaskURL(req.URL, menu.SelectedBrowser)
		req.RuntimeRoot = runtimeRoot
		req.BrowserPath = menu.FirefoxExecutablePath
		req.BrowserInstallDir = menu.FirefoxInstallRoot
		req.DriverDir = menu.PlaywrightDriverDir
		if strings.TrimSpace(req.DownloadRoot) == "" {
			req.DownloadRoot = downloadDir
		}
		if strings.TrimSpace(req.OutputDir) == "" {
			req.OutputDir = downloadDir
		}
		return req
	})
	if changed > 0 {
		a.setStatus("updated browser settings for %d unfinished tasks", changed)
		a.post(msgRefreshTasks)
	}
	return changed
}

func (a *frontendApp) pickDownloadDir() {
	path, err := browseFolderDialog(a.hwnd, "Select download directory", a.downloadDir)
	if err != nil {
		a.setStatus("select download directory failed: %v", err)
		return
	}
	if strings.TrimSpace(path) == "" {
		a.setStatus("download directory unchanged")
		return
	}
	a.mu.Lock()
	a.downloadDir = path
	a.mu.Unlock()
	a.persistFrontendState()
	a.setStatus("download directory set: %s", path)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) cycleConcurrency() {
	a.mu.Lock()
	switch a.concurrency {
	case 1:
		a.concurrency = 2
	case 2:
		a.concurrency = 4
	case 4:
		a.concurrency = 8
	default:
		a.concurrency = 1
	}
	concurrency := a.concurrency
	a.mu.Unlock()
	a.todo.SetConcurrencyLimit(concurrency)
	a.updateConcurrencyButton(concurrency)
	a.persistFrontendState()
	a.setStatus("concurrency set to %d", concurrency)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) promptProgressDelay() {
	a.promptProgressDelayDialog()
}

func (a *frontendApp) refreshAdblockRules() {
	rulePath := filepath.Join(a.workspaceRoot, "adblock", "AWAvenue-Ads-Rule.txt")
	data, err := os.ReadFile(rulePath)
	if err != nil {
		a.setStatus("load adblock rules failed: %v", err)
		return
	}
	lines := strings.Count(string(data), "\n")
	a.mu.Lock()
	a.adblockInfo = fmt.Sprintf("loaded %d lines from %s", lines, rulePath)
	a.mu.Unlock()
	a.setStatus("adblock rules updated: %s", rulePath)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) clearCompletedTasks() {
	cleared := a.todo.ClearFinished()
	a.setStatus("cleared %d finished tasks", cleared)
	a.post(msgRefreshTasks)
	a.post(msgRefreshInfo)
}

func (a *frontendApp) updateConcurrencyButton(value int) {
	if a.concurrencyBtn == 0 {
		return
	}
	label := fmt.Sprintf("\u5e76\u53d1 x%d", value)
	text, _ := utf16Ptr(label)
	procSetWindowTextW.Call(uintptr(a.concurrencyBtn), uintptr(unsafe.Pointer(text)))
}

func (a *frontendApp) currentProgressDelay() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.progressDelay <= 0 {
		return 80 * time.Millisecond
	}
	return a.progressDelay
}

func (a *frontendApp) currentProgressDelayMS() int {
	return int(a.currentProgressDelay() / time.Millisecond)
}

func (a *frontendApp) addPendingTask() {
	url := strings.TrimSpace(getControlText(a.urlEdit))
	if url == "" {
		a.setStatus("URL cannot be empty")
		return
	}
	if strings.Contains(strings.ToLower(url), "myreadingmanga.info") {
		msg := "\u6682\u4e0d\u652f\u6301\u6b64\u7ad9\u70b9"
		a.setStatus("%s: %s", msg, url)
		msgBox(a.hwnd, msg+"\r\n\r\n"+url, "Unsupported Site")
		return
	}
	a.mu.RLock()
	menu := a.menuState
	downloadDir := a.downloadDir
	req := tasks.BrowserLaunchRequest{
		URL:               url,
		BrowserType:       browserTypeForTaskURL(url, menu.SelectedBrowser),
		RuntimeRoot:       a.paths.Root,
		BrowserPath:       menu.FirefoxExecutablePath,
		BrowserInstallDir: menu.FirefoxInstallRoot,
		DriverDir:         menu.PlaywrightDriverDir,
		DownloadRoot:      downloadDir,
		OutputDir:         downloadDir,
		Headless:          true,
	}
	a.mu.RUnlock()
	if duplicate, ok := a.todo.FindDuplicate(req); ok {
		if !confirmBox(a.hwnd, fmt.Sprintf("A similar task already exists.\r\n\r\nExisting: %s\r\nURL: %s\r\n\r\nContinue adding anyway?", duplicate.ID, req.URL), "Duplicate Task") {
			a.setStatus("duplicate task canceled: %s", req.URL)
			return
		}
	}
	a.clearURLInput()
	a.setStatus("starting task...")
	a.todo.SetConcurrencyLimit(a.currentConcurrency())
	go func() {
		item, err := a.todo.RunImmediately(req, nil)
		if err != nil {
			a.setStatus("task %s failed: %v", item.ID, err)
		} else {
			a.scrollTaskBoardToTop()
			a.setStatus("task %s completed", item.ID)
		}
		a.post(msgRefreshTasks)
		a.post(msgRefreshInfo)
	}()
}

func (a *frontendApp) clearURLInput() {
	if a.urlEdit == 0 {
		return
	}
	empty, _ := utf16Ptr("")
	procSetWindowTextW.Call(uintptr(a.urlEdit), uintptr(unsafe.Pointer(empty)))
	setCueBanner(a.urlEdit, "输入漫画页面 URL")
}

func (a *frontendApp) scrollTaskBoardToTop() {
	a.mu.Lock()
	a.taskScrollY = 0
	a.mu.Unlock()
	if a.taskBoard != 0 {
		procSetScrollPos.Call(uintptr(a.taskBoard), 1, 0, 1)
		procInvalidateRect.Call(uintptr(a.taskBoard), 0, 1)
	}
}

func browserTypeForTaskURL(rawURL, selected string) string {
	_, _ = rawURL, selected
	return string(projectruntime.BrowserTypeFirefox)
}

func (a *frontendApp) startPendingAsync() {
	go func() {
		concurrency := a.currentConcurrency()
		a.todo.SetConcurrencyLimit(concurrency)
		if a.todo.BrowserRunning(a.workspaceRoot) {
			a.setStatus("browser is running, please close the current browser window")
			if err := a.todo.WaitForBrowserClose(a.workspaceRoot, 250*time.Millisecond); err != nil {
				a.setStatus("wait for browser close failed: %v", err)
				return
			}
		}
		a.setStatus("starting unfinished tasks with concurrency %d...", concurrency)
		items, err := a.todo.StartAllUnfinishedWithConcurrency(a.workspaceRoot, concurrency, nil)
		if err != nil {
			a.setStatus("start tasks failed: %v", err)
			return
		}
		a.setStatus("task run complete, processed %d items", len(items))
		a.post(msgRefreshTasks)
		a.post(msgRefreshInfo)
	}()
}

func (a *frontendApp) currentConcurrency() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.concurrency <= 0 {
		return 1
	}
	return a.concurrency
}

func (a *frontendApp) importLegacyHistoryAsync() {
	go func() {
		initial := a.defaultLegacyHistoryImportPath()
		path, err := openFileDialog(a.hwnd, "Import legacy history", "JSON Files (*.json)\x00*.json\x00All Files (*.*)\x00*.*\x00\x00", initial)
		if err != nil {
			a.setStatus("select legacy history failed: %v", err)
			return
		}
		path = strings.TrimSpace(path)
		if path == "" {
			a.setStatus("legacy history import canceled")
			return
		}
		state, err := projectruntime.LoadLegacyComicDownloaderState(path)
		if err != nil {
			a.setStatus("import legacy history failed: %v", err)
			return
		}
		added, duplicates := a.todo.PreviewLegacyComicDownloaderState(state)
		preview := fmt.Sprintf("Import preview:\r\n\r\nNew tasks: %d\r\nDuplicates skipped: %d\r\n\r\nContinue importing from:\r\n%s", added, duplicates, path)
		if !confirmBox(a.hwnd, preview, "Import History") {
			a.setStatus("legacy history import canceled by user")
			return
		}
		a.setActionProgress(0.15, "\u9009\u62e9\u5386\u53f2")
		a.setStatus("importing legacy history from %s...", path)
		a.setActionProgress(0.55, "\u52a0\u8f7d\u5386\u53f2")
		count := a.todo.ImportLegacyComicDownloaderState(state)
		if count == 0 {
			a.setActionProgress(1, "\u5b8c\u6210")
			a.setStatus("no legacy history imported from %s", path)
			msgBox(a.hwnd, "No legacy history was imported.\r\n\r\n"+path, "Import History")
			return
		}
		a.setActionProgress(1, "\u5b8c\u6210")
		a.setStatus("imported %d legacy tasks from %s", count, path)
		msgBox(a.hwnd, fmt.Sprintf("Imported %d legacy tasks.\r\n\r\n%s", count, path), "Import History")
		a.post(msgRefreshTasks)
		a.post(msgRefreshInfo)
	}()
}

func (a *frontendApp) setActionProgress(value float64, text string) {
	a.mu.Lock()
	a.actionProgressValue = clamp01(value)
	a.actionProgressText = text
	a.mu.Unlock()
	a.post(msgRefreshAction)
}

func (a *frontendApp) refreshActionProgress() {
	if a.actionProgress == 0 {
		return
	}
	procInvalidateRect.Call(uintptr(a.actionProgress), 0, 1)
}

func (a *frontendApp) actionProgressSnapshot() (float64, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.actionProgressValue, a.actionProgressText
}

func (a *frontendApp) defaultBrowserInstallRoot() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if strings.TrimSpace(a.menuState.FirefoxInstallRoot) != "" {
		return a.menuState.FirefoxInstallRoot
	}
	return projectruntime.DefaultFirefoxInstallDir(a.paths.Root)
}

func (a *frontendApp) defaultPlaywrightRoot() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if strings.TrimSpace(a.menuState.FirefoxInstallRoot) != "" {
		return a.menuState.FirefoxInstallRoot
	}
	if strings.TrimSpace(a.menuState.PlaywrightDriverDir) != "" {
		return filepath.Dir(a.menuState.PlaywrightDriverDir)
	}
	return projectruntime.DefaultFirefoxInstallDir(a.paths.Root)
}

func normalizeBrowserMenuState(menu ui.BrowserMenuState) ui.BrowserMenuState {
	if strings.TrimSpace(menu.FirefoxInstallRoot) != "" {
		if next, _, err := menu.WithPlaywrightRoot(menu.FirefoxInstallRoot); err == nil {
			menu = next
		}
	}
	if strings.TrimSpace(menu.PlaywrightDriverDir) != "" {
		if next, _, err := menu.WithPlaywrightRoot(menu.PlaywrightDriverDir); err == nil {
			return next
		}
		if driverDir, err := ui.ResolvePlaywrightDriverDir(menu.PlaywrightDriverDir); err == nil {
			menu = menu.WithPlaywrightDriverDir(driverDir)
		}
	}
	return menu
}

func (a *frontendApp) defaultLegacyHistoryImportPath() string {
	candidates := []string{
		strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_STATE_PATH")),
		`D:\tools\crawler_NH\20260410_Final01\runtime\comic_downloader_state.json`,
		projectruntime.ResolveLegacyComicDownloaderStatePath(a.workspaceRoot),
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return projectruntime.ResolveLegacyComicDownloaderStatePath(a.workspaceRoot)
}

func browserTypeLabel(browserType projectruntime.BrowserType) string {
	switch browserType {
	case projectruntime.BrowserTypeFirefox:
		return "Firefox"
	default:
		return string(browserType)
	}
}

func (a *frontendApp) post(msg uint32) {
	if a.hwnd != 0 {
		procPostMessageW.Call(uintptr(a.hwnd), uintptr(msg), 0, 0)
	}
}

func (a *frontendApp) setStatus(format string, args ...any) {
	a.mu.Lock()
	a.status = fmt.Sprintf(format, args...)
	a.mu.Unlock()
	a.post(msgRefreshStatus)
}

func (a *frontendApp) refreshStatus() {
	if a.statusText == 0 {
		return
	}
	procSetWindowTextW.Call(uintptr(a.statusText), uintptr(unsafe.Pointer(utf16PtrMust(a.statusSnapshot()))))
}

func (a *frontendApp) refreshTaskList() {
	if a.taskBoard == 0 {
		return
	}
	taskBoardRefresh(a.taskBoard)
}

func (a *frontendApp) refreshInfo() {
	return
}

func (a *frontendApp) refreshAllUI() {
	a.refreshStatus()
	a.refreshTaskList()
	a.refreshInfo()
}

func (a *frontendApp) statusSnapshot() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func msgBox(hwnd HWND, text, caption string) {
	textPtr, _ := utf16Ptr(text)
	captionPtr, _ := utf16Ptr(caption)
	procMessageBoxW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(captionPtr)), MB_OK)
}

func confirmBox(hwnd HWND, text, caption string) bool {
	textPtr, _ := utf16Ptr(text)
	captionPtr, _ := utf16Ptr(caption)
	r, _, _ := procMessageBoxW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(captionPtr)), MB_YESNO|MB_ICONQUESTION)
	return r == 6
}

func getControlText(hwnd HWND) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLen.Call(uintptr(hwnd))
	buf := make([]uint16, int(n)+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return strings.TrimRight(string(utf16.Decode(buf)), "\x00")
}

func openFileDialog(hwnd HWND, title, filter, initialPath string) (string, error) {
	fileBuf := make([]uint16, 32768)
	initialDir := initialPath
	if strings.TrimSpace(initialDir) != "" {
		initialDir = filepath.Dir(initialDir)
	}
	var initialDirPtr *uint16
	if strings.TrimSpace(initialDir) != "" {
		initialDirPtr, _ = utf16Ptr(initialDir)
	}
	titlePtr, _ := utf16Ptr(title)
	filterPtr, _ := utf16Ptr(filter)
	ofn := OPENFILENAMEW{
		LStructSize:     uint32(unsafe.Sizeof(OPENFILENAMEW{})),
		HwndOwner:       hwnd,
		LpstrFilter:     filterPtr,
		LpstrFile:       &fileBuf[0],
		NMaxFile:        uint32(len(fileBuf)),
		LpstrTitle:      titlePtr,
		LpstrInitialDir: initialDirPtr,
		Flags:           OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST,
	}
	r, _, err := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return "", err
	}
	return strings.TrimRight(string(utf16.Decode(fileBuf)), "\x00"), nil
}

func utf16Ptr(s string) (*uint16, error) {
	return syscall.UTF16PtrFromString(s)
}

func utf16PtrMust(s string) *uint16 {
	p, err := utf16Ptr(s)
	if err != nil {
		return nil
	}
	return p
}

func loadAppIcon(iconPath string) (uintptr, error) {
	iconPath = filepath.Clean(strings.TrimSpace(iconPath))
	if iconPath == "" {
		return 0, fmt.Errorf("icon path is empty")
	}
	if _, err := os.Stat(iconPath); err != nil {
		return 0, err
	}
	pathPtr, err := utf16Ptr(iconPath)
	if err != nil {
		return 0, err
	}
	handle, _, callErr := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		IMAGE_ICON,
		0,
		0,
		LR_LOADFROMFILE|LR_DEFAULTSIZE|LR_SHARED,
	)
	if handle == 0 {
		if callErr != syscall.Errno(0) {
			return 0, fmt.Errorf("LoadImageW: %w", callErr)
		}
		return 0, fmt.Errorf("LoadImageW failed for %s", iconPath)
	}
	return handle, nil
}

func setExplicitAppUserModelID(appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil
	}
	idPtr, err := utf16Ptr(appID)
	if err != nil {
		return err
	}
	hr, _, callErr := procSetCurrentProcessExplicitAppUserModelID.Call(uintptr(unsafe.Pointer(idPtr)))
	if hr != 0 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("SetCurrentProcessExplicitAppUserModelID: %w", callErr)
		}
		return fmt.Errorf("SetCurrentProcessExplicitAppUserModelID failed")
	}
	return nil
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func executableWorkspaceRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_WORKSPACE_ROOT")); override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		return filepath.Clean(abs), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func ensureThemeResources() error {
	themeOnce.Do(func() {
		bgHandle, _, _ := procCreateSolidBrush.Call(uintptr(rgb(18, 22, 28)))
		panelHandle, _, _ := procCreateSolidBrush.Call(uintptr(rgb(28, 34, 42)))
		editHandle, _, _ := procCreateSolidBrush.Call(uintptr(rgb(36, 42, 52)))
		activeBtnHandle, _, _ := procCreateSolidBrush.Call(uintptr(rgb(66, 112, 76)))
		darkBgBrush = HBRUSH(bgHandle)
		darkPanelBrush = HBRUSH(panelHandle)
		darkEditBrush = HBRUSH(editHandle)
		activeBtnBrush = HBRUSH(activeBtnHandle)
		uiFont = createUIFont(-14, 400, "Segoe UI")
		rowFont = createUIFont(-12, 400, "Segoe UI")
		titleFont = createUIFont(-14, 700, "Segoe UI")
		if darkBgBrush == 0 || darkPanelBrush == 0 || darkEditBrush == 0 || activeBtnBrush == 0 || uiFont == 0 || rowFont == 0 || titleFont == 0 {
			themeInitErr = fmt.Errorf("initialize theme resources failed")
		}
	})
	return themeInitErr
}

func applyDarkWindowChrome(hwnd HWND) {
	if hwnd == 0 {
		return
	}
	if err := procDwmSetWindowAttribute.Find(); err == nil {
		enabled := uint32(1)
		_, _, _ = procDwmSetWindowAttribute.Call(
			uintptr(hwnd),
			uintptr(DWMWA_USE_IMMERSIVE_DARK_MODE),
			uintptr(unsafe.Pointer(&enabled)),
			uintptr(unsafe.Sizeof(enabled)),
		)
		caption := rgb(18, 22, 28)
		text := rgb(230, 236, 242)
		_, _, _ = procDwmSetWindowAttribute.Call(
			uintptr(hwnd),
			uintptr(DWMWA_CAPTION_COLOR),
			uintptr(unsafe.Pointer(&caption)),
			uintptr(unsafe.Sizeof(caption)),
		)
		_, _, _ = procDwmSetWindowAttribute.Call(
			uintptr(hwnd),
			uintptr(DWMWA_TEXT_COLOR),
			uintptr(unsafe.Pointer(&text)),
			uintptr(unsafe.Sizeof(text)),
		)
	}
}

func enableDarkAppMode() {
	if err := procSetPreferredAppMode.Find(); err == nil {
		_, _, _ = procSetPreferredAppMode.Call(uintptr(APPMODE_ALLOWDARK))
	}
	if err := procFlushMenuThemes.Find(); err == nil {
		_, _, _ = procFlushMenuThemes.Call()
	}
}

func createUIFont(height int32, weight int32, face string) HGDIOBJ {
	facePtr, err := utf16Ptr(face)
	if err != nil {
		return 0
	}
	handle, _, _ := procCreateFontW.Call(
		uintptr(height),
		0,
		0,
		0,
		uintptr(weight),
		0,
		0,
		0,
		1,
		0,
		0,
		5,
		0,
		uintptr(unsafe.Pointer(facePtr)),
	)
	return HGDIOBJ(handle)
}

func setControlFont(hwnd HWND, font HGDIOBJ) {
	if hwnd == 0 || font == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hwnd), WM_SETFONT, uintptr(font), 1)
}
