//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	gdiplusDLL                   = syscall.NewLazyDLL("gdiplus.dll")
	procGdiplusStartup           = gdiplusDLL.NewProc("GdiplusStartup")
	procGdiplusShutdown          = gdiplusDLL.NewProc("GdiplusShutdown")
	procGdipCreateBitmapFromFile = gdiplusDLL.NewProc("GdipCreateBitmapFromFile")
	procGdipCreateFromHDC        = gdiplusDLL.NewProc("GdipCreateFromHDC")
	procSaveDC                   = syscall.NewLazyDLL("gdi32.dll").NewProc("SaveDC")
	procRestoreDC                = syscall.NewLazyDLL("gdi32.dll").NewProc("RestoreDC")
	procCreateRoundRectRgn       = syscall.NewLazyDLL("gdi32.dll").NewProc("CreateRoundRectRgn")
	procSelectClipRgn            = syscall.NewLazyDLL("gdi32.dll").NewProc("SelectClipRgn")
	procFillRgn                  = syscall.NewLazyDLL("gdi32.dll").NewProc("FillRgn")
	procFrameRgn                 = syscall.NewLazyDLL("gdi32.dll").NewProc("FrameRgn")
	procGdipDeleteGraphics       = gdiplusDLL.NewProc("GdipDeleteGraphics")
	procGdipDisposeImage         = gdiplusDLL.NewProc("GdipDisposeImage")
	procGdipDrawImageRectI       = gdiplusDLL.NewProc("GdipDrawImageRectI")
)

type GdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread uint32
	SuppressExternalCodecs   uint32
}

type thumbnailCacheEntry struct {
	image    uintptr
	lastUsed time.Time
}

var (
	thumbnailMu    sync.Mutex
	thumbnailCache = make(map[string]thumbnailCacheEntry)
	gdiplusToken   uintptr
	gdiplusOnce    sync.Once
	gdiplusErr     error
)

func ensureGDIPlus() error {
	gdiplusOnce.Do(func() {
		input := GdiplusStartupInput{
			GdiplusVersion: 1,
		}
		status, _, callErr := procGdiplusStartup.Call(
			uintptr(unsafe.Pointer(&gdiplusToken)),
			uintptr(unsafe.Pointer(&input)),
			0,
		)
		if status != 0 {
			if callErr != syscall.Errno(0) {
				gdiplusErr = fmt.Errorf("GdiplusStartup: %w", callErr)
			} else {
				gdiplusErr = fmt.Errorf("GdiplusStartup failed with status %d", status)
			}
			return
		}
	})
	return gdiplusErr
}

func shutdownGDIPlus() {
	if gdiplusToken == 0 {
		return
	}
	procGdiplusShutdown.Call(gdiplusToken)
	gdiplusToken = 0
}

func shutdownTaskThumbnailCache() {
	thumbnailMu.Lock()
	defer thumbnailMu.Unlock()
	for path, entry := range thumbnailCache {
		if entry.image != 0 {
			procGdipDisposeImage.Call(entry.image)
		}
		delete(thumbnailCache, path)
	}
}

func taskThumbnailImage(path string) (uintptr, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return 0, fmt.Errorf("thumbnail path is empty")
	}
	if err := ensureGDIPlus(); err != nil {
		return 0, err
	}

	thumbnailMu.Lock()
	if entry, ok := thumbnailCache[path]; ok && entry.image != 0 {
		entry.lastUsed = time.Now()
		thumbnailCache[path] = entry
		imageHandle := entry.image
		thumbnailMu.Unlock()
		return imageHandle, nil
	}
	thumbnailMu.Unlock()

	if _, err := os.Stat(path); err != nil {
		return 0, err
	}
	pathPtr, err := utf16Ptr(path)
	if err != nil {
		return 0, err
	}
	var imageHandle uintptr
	status, _, callErr := procGdipCreateBitmapFromFile.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&imageHandle)),
	)
	if status != 0 || imageHandle == 0 {
		if callErr != syscall.Errno(0) {
			return 0, fmt.Errorf("GdipCreateBitmapFromFile: %w", callErr)
		}
		return 0, fmt.Errorf("GdipCreateBitmapFromFile failed with status %d", status)
	}

	thumbnailMu.Lock()
	defer thumbnailMu.Unlock()
	if len(thumbnailCache) >= 128 {
		var oldestPath string
		var oldestTime time.Time
		for cachedPath, entry := range thumbnailCache {
			if oldestPath == "" || entry.lastUsed.Before(oldestTime) {
				oldestPath = cachedPath
				oldestTime = entry.lastUsed
			}
		}
		if oldestPath != "" {
			if oldEntry := thumbnailCache[oldestPath]; oldEntry.image != 0 {
				procGdipDisposeImage.Call(oldEntry.image)
			}
			delete(thumbnailCache, oldestPath)
		}
	}
	thumbnailCache[path] = thumbnailCacheEntry{image: imageHandle, lastUsed: time.Now()}
	return imageHandle, nil
}

func drawTaskThumbnail(hdc HDC, rect RECT, path string, fallbackBrush HBRUSH) {
	if saved, _, _ := procSaveDC.Call(uintptr(hdc)); saved != 0 {
		defer procRestoreDC.Call(uintptr(hdc), saved)
	}
	if fallbackBrush != 0 {
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rect)), uintptr(fallbackBrush))
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	imageHandle, err := taskThumbnailImage(path)
	if err != nil {
		return
	}
	var graphics uintptr
	status, _, callErr := procGdipCreateFromHDC.Call(uintptr(hdc), uintptr(unsafe.Pointer(&graphics)))
	if status != 0 || graphics == 0 {
		_ = callErr
		return
	}
	defer procGdipDeleteGraphics.Call(graphics)
	outerRect := RECT{Left: rect.Left + 1, Top: rect.Top + 1, Right: rect.Right - 1, Bottom: rect.Bottom - 1}
	innerRect := RECT{Left: rect.Left + 4, Top: rect.Top + 4, Right: rect.Right - 4, Bottom: rect.Bottom - 4}
	if outerRect.Right <= outerRect.Left || outerRect.Bottom <= outerRect.Top {
		return
	}
	if outerRegion, _, _ := procCreateRoundRectRgn.Call(
		uintptr(outerRect.Left),
		uintptr(outerRect.Top),
		uintptr(outerRect.Right),
		uintptr(outerRect.Bottom),
		uintptr(18),
		uintptr(18),
	); outerRegion != 0 {
		defer procDeleteObject.Call(outerRegion)
		if fallbackBrush != 0 {
			procFillRgn.Call(uintptr(hdc), outerRegion, uintptr(fallbackBrush))
		}
		borderBrush, _, _ := procCreateSolidBrush.Call(uintptr(rgb(152, 176, 198)))
		if borderBrush != 0 {
			procFrameRgn.Call(uintptr(hdc), outerRegion, borderBrush, uintptr(2), uintptr(2))
			procDeleteObject.Call(borderBrush)
		}
	}
	if innerRect.Right <= innerRect.Left || innerRect.Bottom <= innerRect.Top {
		return
	}
	innerRegion, _, _ := procCreateRoundRectRgn.Call(
		uintptr(innerRect.Left),
		uintptr(innerRect.Top),
		uintptr(innerRect.Right),
		uintptr(innerRect.Bottom),
		uintptr(12),
		uintptr(12),
	)
	if innerRegion != 0 {
		defer procDeleteObject.Call(innerRegion)
		procSelectClipRgn.Call(uintptr(hdc), innerRegion)
	}
	left := innerRect.Left + 1
	top := innerRect.Top + 1
	right := innerRect.Right - 1
	bottom := innerRect.Bottom - 1
	if right <= left || bottom <= top {
		return
	}
	procGdipDrawImageRectI.Call(
		graphics,
		imageHandle,
		uintptr(left),
		uintptr(top),
		uintptr(right-left),
		uintptr(bottom-top),
	)
	if innerRegion != 0 {
		procSelectClipRgn.Call(uintptr(hdc), 0)
	}
}
