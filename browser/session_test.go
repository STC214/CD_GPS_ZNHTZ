package browser

import (
	"path/filepath"
	"testing"
)

func TestLaunchDataAndContextData(t *testing.T) {
	m := NewFirefoxMiddleware("https://example.com/reader")
	m = m.WithRuntimeRoot(filepath.Clean(filepath.FromSlash("F:/Project/comic_downloader_GO_Playwright_stealth/runtime")))
	m = m.WithBrowserPath(`C:\Program Files\Mozilla Firefox\firefox.exe`)
	m = m.WithHeadless(true)

	launch := m.LaunchData(BrowserSessionOptions{Headless: HeadlessPtr(true)})
	if launch.ExecutablePath == "" {
		t.Fatalf("LaunchData().ExecutablePath is empty")
	}
	if got, want := launch.ExecutablePath, filepath.Clean(filepath.FromSlash(`C:\Program Files\Mozilla Firefox\firefox.exe`)); got != want {
		t.Fatalf("LaunchData().ExecutablePath = %q, want %q", got, want)
	}
	if !launch.Headless {
		t.Fatalf("LaunchData().Headless = false, want true")
	}

	context := m.ContextData(BrowserSessionOptions{Headless: HeadlessPtr(false)})
	if got, want := context.BaseURL, "https://example.com/reader"; got != want {
		t.Fatalf("ContextData().BaseURL = %q, want %q", got, want)
	}
}
