package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
)

func TestBrowserProfileMiddlewareClosesBrowserAndCopiesProfile(t *testing.T) {
	workspace := t.TempDir()
	appData := filepath.Join(workspace, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))

	sourceDir := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "aocfvl86.default-default-3")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appData, "Mozilla", "Firefox"), 0o755); err != nil {
		t.Fatalf("create firefox dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appData, "Mozilla", "Firefox", "profiles.ini"), []byte(`
[Profile0]
Name=default
IsRelative=1
Path=Profiles/aocfvl86.default-default-3
Default=1
`), 0o644); err != nil {
		t.Fatalf("write profiles.ini: %v", err)
	}
	t.Setenv("COMIC_FIREFOX_PROFILE_SOURCE_DIR", sourceDir)
	if err := os.WriteFile(filepath.Join(sourceDir, "prefs.js"), []byte("source-profile"), 0o644); err != nil {
		t.Fatalf("write source prefs.js: %v", err)
	}

	paths := runtime.NewPaths(workspace)
	if err := paths.Ensure(); err != nil {
		t.Fatalf("paths.Ensure() error = %v", err)
	}

	release, err := runtime.AcquireBrowserSessionLock(paths.Root)
	if err != nil {
		t.Fatalf("AcquireBrowserSessionLock() error = %v", err)
	}

	middleware := NewBrowserProfileMiddleware(workspace)
	done := make(chan BrowserProfileRefreshResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := middleware.CloseCurrentBrowserAndCopyFirefoxProfile(10 * time.Millisecond)
		done <- result
		errCh <- err
	}()

	select {
	case <-done:
		t.Fatal("refresh returned before browser lock was released")
	case <-time.After(50 * time.Millisecond):
	}

	if err := release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	var result BrowserProfileRefreshResult
	select {
	case result = <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh did not finish after browser lock release")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("CloseCurrentBrowserAndCopyFirefoxProfile() error = %v", err)
	}
	if result.TargetProfileDir != filepath.Join(workspace, "runtime", "browser-profiles", "baseline-userdata") {
		t.Fatalf("TargetProfileDir = %q", result.TargetProfileDir)
	}
	got, err := os.ReadFile(filepath.Join(result.TargetProfileDir, "prefs.js"))
	if err != nil {
		t.Fatalf("read target prefs.js: %v", err)
	}
	if string(got) != "source-profile" {
		t.Fatalf("copied prefs.js = %q, want source-profile", string(got))
	}
}
