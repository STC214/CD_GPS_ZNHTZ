package ui

import (
	"errors"
	"fmt"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
)

// BrowserProfileMiddleware coordinates browser close-and-copy actions for the frontend.
type BrowserProfileMiddleware struct {
	WorkspaceRoot string
}

// BrowserProfileRefreshResult describes one browser profile refresh.
type BrowserProfileRefreshResult struct {
	SourceProfileDir string `json:"sourceProfileDir"`
	TargetProfileDir string `json:"targetProfileDir"`
	BrowserClosed    bool   `json:"browserClosed"`
}

// NewBrowserProfileMiddleware builds a browser profile middleware for one workspace.
func NewBrowserProfileMiddleware(workspaceRoot string) BrowserProfileMiddleware {
	return BrowserProfileMiddleware{WorkspaceRoot: workspaceRoot}
}

// FirefoxWorkingProfileDir returns the project-owned working profile directory used by tasks.
func (m BrowserProfileMiddleware) FirefoxWorkingProfileDir() string {
	return runtime.NewBrowserProfileManager(m.WorkspaceRoot).MotherProfileDir(runtime.BrowserTypeFirefox)
}

// CloseCurrentBrowserAndCopyFirefoxProfile waits for the browser session to end, then refreshes the working profile.
func (m BrowserProfileMiddleware) CloseCurrentBrowserAndCopyFirefoxProfile(pollInterval time.Duration) (BrowserProfileRefreshResult, error) {
	manager := runtime.NewBrowserProfileManager(m.WorkspaceRoot)
	sourceDir, err := manager.SourceProfileDir(runtime.BrowserTypeFirefox)
	if err != nil {
		return BrowserProfileRefreshResult{}, err
	}
	return m.CloseCurrentBrowserAndCopyFirefoxProfileFromSource(sourceDir, pollInterval)
}

// CloseCurrentBrowserAndCopyFirefoxProfileFromSource waits for the browser session to end, then refreshes the working profile from sourceDir.
func (m BrowserProfileMiddleware) CloseCurrentBrowserAndCopyFirefoxProfileFromSource(sourceDir string, pollInterval time.Duration) (BrowserProfileRefreshResult, error) {
	paths := runtime.NewPaths(m.WorkspaceRoot)
	if err := paths.Ensure(); err != nil {
		return BrowserProfileRefreshResult{}, err
	}
	runtimeRoot := paths.Root
	browserBusy := runtime.BrowserSessionLocked(runtimeRoot)
	if browserBusy {
		if err := runtime.WaitForBrowserSessionUnlock(runtimeRoot, pollInterval); err != nil {
			return BrowserProfileRefreshResult{}, err
		}
	}

	manager := runtime.NewBrowserProfileManager(m.WorkspaceRoot)
	result, err := manager.RefreshProjectProfileFromSource(runtime.BrowserTypeFirefox, sourceDir)
	if err != nil {
		return BrowserProfileRefreshResult{}, err
	}
	return BrowserProfileRefreshResult{
		SourceProfileDir: result.SourceProfileDir,
		TargetProfileDir: result.TargetProfileDir,
		BrowserClosed:    browserBusy || !runtime.BrowserSessionLocked(runtimeRoot),
	}, nil
}

// CloseCurrentBrowserAndCopyFirefoxProfileNow refreshes the working profile immediately if no browser is running.
func (m BrowserProfileMiddleware) CloseCurrentBrowserAndCopyFirefoxProfileNow() (BrowserProfileRefreshResult, error) {
	if runtime.BrowserSessionLocked(runtime.NewPaths(m.WorkspaceRoot).Root) {
		return BrowserProfileRefreshResult{}, errors.New("browser is running; close the current browser window before copying the profile")
	}
	return m.CloseCurrentBrowserAndCopyFirefoxProfile(10 * time.Millisecond)
}

// ErrBrowserBusy is returned when the browser is still open and the caller requested an immediate refresh.
var ErrBrowserBusy = errors.New("browser is running")

// BusyError describes a browser busy state with a clearer message for UI prompts.
func BusyError() error {
	return fmt.Errorf("%w; close the current browser window before copying the profile", ErrBrowserBusy)
}
