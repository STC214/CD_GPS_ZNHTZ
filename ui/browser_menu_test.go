package ui

import (
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
