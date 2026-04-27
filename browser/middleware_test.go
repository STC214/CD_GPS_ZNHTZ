package browser

import (
	"path/filepath"
	"testing"
)

func TestFirefoxMiddlewareLaunchSpecAndPayload(t *testing.T) {
	m := NewFirefoxMiddleware("https://example.com/page")
	m = m.WithRuntimeRoot("F:/Project/comic_downloader_GO_Playwright_stealth/runtime")
	m = m.WithDownloadRoot("F:/Project/comic_downloader_GO_Playwright_stealth/downloads")
	m = m.WithOutputDir("F:/Project/comic_downloader_GO_Playwright_stealth/download")
	m = m.WithProfileDir("C:/Users/stc52/AppData/Roaming/Mozilla/Firefox/Profiles/q6nkoa5l.default-default-1")
	m = m.WithUserDataDir("F:/Project/comic_downloader_GO_Playwright_stealth/runtime/browser-profiles/firefox")
	m = m.WithUserAgent("Mozilla/5.0 test agent")
	m = m.WithFirefoxUserPrefs(map[string]any{"browser.tabs.warnOnClose": false})
	m = m.WithBrowserPath(`C:\Program Files\Mozilla Firefox\firefox.exe`)
	m = m.WithHeadless(false)
	m = m.WithAdblock(true)

	if got := m.URL(); got != "https://example.com/page" {
		t.Fatalf("URL() = %q, want %q", got, "https://example.com/page")
	}

	spec := m.LaunchSpec(BrowserSessionOptions{Headless: HeadlessPtr(false)})
	if spec.BrowserType != BrowserTypeFirefox {
		t.Fatalf("LaunchSpec().BrowserType = %q, want %q", spec.BrowserType, BrowserTypeFirefox)
	}
	if spec.BrowserPath != filepath.Clean(filepath.FromSlash(`C:\Program Files\Mozilla Firefox\firefox.exe`)) {
		t.Fatalf("LaunchSpec().BrowserPath = %q", spec.BrowserPath)
	}
	if spec.StealthScript.Path != filepath.Clean(filepath.FromSlash("F:/Project/comic_downloader_GO_Playwright_stealth/runtime/firefox_stealth.js")) {
		t.Fatalf("LaunchSpec().StealthScript.Path = %q", spec.StealthScript.Path)
	}
	if spec.ProfileDir != filepath.Clean(filepath.FromSlash("C:/Users/stc52/AppData/Roaming/Mozilla/Firefox/Profiles/q6nkoa5l.default-default-1")) {
		t.Fatalf("LaunchSpec().ProfileDir = %q", spec.ProfileDir)
	}
	if spec.UserDataDir != filepath.Clean(filepath.FromSlash("F:/Project/comic_downloader_GO_Playwright_stealth/runtime/browser-profiles/firefox")) {
		t.Fatalf("LaunchSpec().UserDataDir = %q", spec.UserDataDir)
	}
	if spec.UserAgent != "Mozilla/5.0 test agent" {
		t.Fatalf("LaunchSpec().UserAgent = %q", spec.UserAgent)
	}
	if spec.FirefoxUserPrefs["browser.tabs.warnOnClose"] != false {
		t.Fatalf("LaunchSpec().FirefoxUserPrefs = %#v", spec.FirefoxUserPrefs)
	}
	if spec.Headless {
		t.Fatalf("LaunchSpec().Headless = true, want false")
	}

	payload := m.Payload(BrowserSessionOptions{Headless: HeadlessPtr(true)})
	if payload.URL != "https://example.com/page" {
		t.Fatalf("Payload().URL = %q", payload.URL)
	}
	if payload.RuntimeRoot != filepath.Clean(filepath.FromSlash("F:/Project/comic_downloader_GO_Playwright_stealth/runtime")) {
		t.Fatalf("Payload().RuntimeRoot = %q", payload.RuntimeRoot)
	}
	if payload.BrowserType != string(BrowserTypeFirefox) {
		t.Fatalf("Payload().BrowserType = %q", payload.BrowserType)
	}
	if payload.ProfileDir != filepath.Clean(filepath.FromSlash("C:/Users/stc52/AppData/Roaming/Mozilla/Firefox/Profiles/q6nkoa5l.default-default-1")) {
		t.Fatalf("Payload().ProfileDir = %q", payload.ProfileDir)
	}
	if payload.UserAgent != "Mozilla/5.0 test agent" {
		t.Fatalf("Payload().UserAgent = %q", payload.UserAgent)
	}
}
