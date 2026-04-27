package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBrowserMenuStateIncludesFirefoxPaths(t *testing.T) {
	menu := DefaultBrowserMenuState()
	if menu.SelectedBrowser != "firefox" {
		t.Fatalf("SelectedBrowser = %q, want firefox", menu.SelectedBrowser)
	}
	if menu.FirefoxExecutablePath != `C:\Program Files\Mozilla Firefox\firefox.exe` {
		t.Fatalf("FirefoxExecutablePath = %q, want system Firefox path", menu.FirefoxExecutablePath)
	}
	wantFirefoxInstall := filepath.Clean(`runtime/playwright-browsers/firefox`)
	if menu.FirefoxInstallRoot != wantFirefoxInstall {
		t.Fatalf("FirefoxInstallRoot = %q, want %q", menu.FirefoxInstallRoot, wantFirefoxInstall)
	}
}

func TestResolvePlaywrightRootFindsFirefoxAndDriver(t *testing.T) {
	root, firefoxPath, driverDir := createPlaywrightRootFixture(t)

	resolved, err := ResolvePlaywrightRoot(root)
	if err != nil {
		t.Fatalf("ResolvePlaywrightRoot() error = %v", err)
	}
	if resolved.FirefoxExecutable != firefoxPath {
		t.Fatalf("FirefoxExecutable = %q, want %q", resolved.FirefoxExecutable, firefoxPath)
	}
	if resolved.PlaywrightDriverDir != driverDir {
		t.Fatalf("PlaywrightDriverDir = %q, want %q", resolved.PlaywrightDriverDir, driverDir)
	}
}

func TestBrowserMenuStateWithPlaywrightRootAppliesAllPaths(t *testing.T) {
	root, firefoxPath, driverDir := createPlaywrightRootFixture(t)

	menu, _, err := (BrowserMenuState{}).WithPlaywrightRoot(root)
	if err != nil {
		t.Fatalf("WithPlaywrightRoot() error = %v", err)
	}
	if menu.FirefoxInstallRoot != filepath.Clean(root) {
		t.Fatalf("FirefoxInstallRoot = %q, want %q", menu.FirefoxInstallRoot, filepath.Clean(root))
	}
	if menu.FirefoxExecutablePath != firefoxPath {
		t.Fatalf("FirefoxExecutablePath = %q, want %q", menu.FirefoxExecutablePath, firefoxPath)
	}
	if menu.PlaywrightDriverDir != driverDir {
		t.Fatalf("PlaywrightDriverDir = %q, want %q", menu.PlaywrightDriverDir, driverDir)
	}
}

func TestResolvePlaywrightDriverDirAcceptsRootOrDriver(t *testing.T) {
	root, _, driverDir := createPlaywrightRootFixture(t)

	fromRoot, err := ResolvePlaywrightDriverDir(root)
	if err != nil {
		t.Fatalf("ResolvePlaywrightDriverDir(root) error = %v", err)
	}
	if fromRoot != driverDir {
		t.Fatalf("ResolvePlaywrightDriverDir(root) = %q, want %q", fromRoot, driverDir)
	}
	fromDriver, err := ResolvePlaywrightDriverDir(driverDir)
	if err != nil {
		t.Fatalf("ResolvePlaywrightDriverDir(driverDir) error = %v", err)
	}
	if fromDriver != driverDir {
		t.Fatalf("ResolvePlaywrightDriverDir(driverDir) = %q, want %q", fromDriver, driverDir)
	}
}

func createPlaywrightRootFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	firefoxPath := filepath.Join(root, "firefox-1497", "firefox", "firefox.exe")
	driverDir := filepath.Join(root, "driver")
	if err := os.MkdirAll(filepath.Dir(firefoxPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(firefoxPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(driverDir, "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driverDir, "node.exe"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, firefoxPath, driverDir
}
