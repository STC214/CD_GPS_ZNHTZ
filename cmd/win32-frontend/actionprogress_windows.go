//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const actionProgressClassName = "ActionProgressControl"

var actionProgressWndProc = syscall.NewCallback(actionProgressWindowProc)

func registerActionProgressClass(hInstance HINSTANCE) error {
	className, err := utf16Ptr(actionProgressClassName)
	if err != nil {
		return err
	}
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   actionProgressWndProc,
		HInstance:     hInstance,
		HCursor:       HCURSOR(cursor),
		HbrBackground: darkBgBrush,
		LpszClassName: className,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW action progress: %w", err)
	}
	return nil
}

func actionProgressWindowProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		actionProgressPaint(hwnd)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func actionProgressPaint(hwnd HWND) {
	var ps PAINTSTRUCT
	hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	bg, _, _ := procCreateSolidBrush.Call(uintptr(rgb(18, 22, 28)))
	defer procDeleteObject.Call(bg)
	panel, _, _ := procCreateSolidBrush.Call(uintptr(rgb(28, 34, 42)))
	defer procDeleteObject.Call(panel)
	fill, _, _ := procCreateSolidBrush.Call(uintptr(rgb(96, 118, 140)))
	defer procDeleteObject.Call(fill)
	done, _, _ := procCreateSolidBrush.Call(uintptr(rgb(84, 138, 92)))
	defer procDeleteObject.Call(done)

	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&ps.RcPaint)), bg)

	if app == nil {
		return
	}
	progress, text := app.actionProgressSnapshot()
	var rc RECT
	if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return
	}

	barLeft := int32(20)
	barTop := int32(4)
	barRight := rc.Right - 160
	if barRight < barLeft+40 {
		barRight = barLeft + 40
	}
	barRect := RECT{Left: barLeft, Top: barTop, Right: barRight, Bottom: barTop + 12}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&barRect)), panel)

	fillRect := barRect
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	fillRect.Right = fillRect.Left + int32(float64(fillRect.Right-fillRect.Left)*progress)
	if fillRect.Right < fillRect.Left+2 {
		fillRect.Right = fillRect.Left + 2
	}
	if progress >= 1 {
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&fillRect)), done)
	} else {
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&fillRect)), fill)
	}

	drawTextLineWithFont(HDC(hdc), uiFont, int(rc.Right)-120, 0, fmt.Sprintf("%d%%", int(progress*100)), rgb(240, 244, 248))
	if text != "" {
		drawTextLineWithFont(HDC(hdc), uiFont, int(barLeft)+int(barRight-barLeft)+12, 0, text, rgb(184, 196, 208))
	}
}
