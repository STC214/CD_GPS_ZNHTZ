package tasks

import (
	"os"
	"path/filepath"
	"testing"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
)

func TestBrowserLaunchRequestBuildsMiddleware(t *testing.T) {
	workspace := t.TempDir()
	projectMother := filepath.Join(workspace, "runtime", "browser-profiles", "firefox")
	if err := os.MkdirAll(projectMother, 0o755); err != nil {
		t.Fatalf("create project mother dir: %v", err)
	}

	req := BrowserLaunchRequest{
		URL:                  "https://example.com",
		Headless:             false,
		RuntimeRoot:          filepath.Join(workspace, "runtime"),
		ProfileDir:           "C:/Users/stc52/AppData/Roaming/Mozilla/Firefox/Profiles/q6nkoa5l.default-default-1",
		UserAgent:            "Mozilla/5.0 test agent",
		FirefoxUserPrefsJSON: `{"browser.tabs.warnOnClose":false}`,
	}
	middleware := req.FirefoxMiddleware()
	spec := middleware.LaunchSpec(req.BrowserOptions())
	if spec.URL != "https://example.com" {
		t.Fatalf("spec.URL = %q", spec.URL)
	}
	if spec.Headless {
		t.Fatalf("spec.Headless = true, want false")
	}
	wantPath := `C:\Program Files\Mozilla Firefox\firefox.exe`
	if spec.BrowserPath != wantPath {
		t.Fatalf("spec.BrowserPath = %q, want %q", spec.BrowserPath, wantPath)
	}
	wantProfile := filepath.Clean("C:/Users/stc52/AppData/Roaming/Mozilla/Firefox/Profiles/q6nkoa5l.default-default-1")
	if spec.ProfileDir != wantProfile {
		t.Fatalf("spec.ProfileDir = %q, want %q", spec.ProfileDir, wantProfile)
	}
	if spec.UserDataDir != wantProfile {
		t.Fatalf("spec.UserDataDir = %q, want %q", spec.UserDataDir, wantProfile)
	}
	if spec.UserAgent != "Mozilla/5.0 test agent" {
		t.Fatalf("spec.UserAgent = %q", spec.UserAgent)
	}
	if spec.FirefoxUserPrefs["browser.tabs.warnOnClose"] != false {
		t.Fatalf("spec.FirefoxUserPrefs = %#v", spec.FirefoxUserPrefs)
	}
}

func TestBrowserLaunchRequestDefaultsUnsupportedBrowserToFirefox(t *testing.T) {
	workspace := t.TempDir()
	req := BrowserLaunchRequest{
		URL:         "https://example.com",
		BrowserType: "unsupported",
		RuntimeRoot: filepath.Join(workspace, "runtime"),
	}
	req = req.Normalize()
	if req.BrowserType != "firefox" {
		t.Fatalf("BrowserType = %q, want firefox", req.BrowserType)
	}
	wantProfile := filepath.Clean(`runtime/browser-profiles/baseline-userdata`)
	if req.ProfileDir != wantProfile {
		t.Fatalf("ProfileDir = %q, want %q", req.ProfileDir, wantProfile)
	}
	if req.UserDataDir != wantProfile {
		t.Fatalf("UserDataDir = %q, want %q", req.UserDataDir, wantProfile)
	}
	wantBrowserPath := `C:\Program Files\Mozilla Firefox\firefox.exe`
	if req.BrowserPath != wantBrowserPath {
		t.Fatalf("BrowserPath = %q, want %q", req.BrowserPath, wantBrowserPath)
	}
}

func TestBrowserLaunchRequestForcesFirefoxForZeriURL(t *testing.T) {
	workspace := t.TempDir()
	req := BrowserLaunchRequest{
		URL:         "https://zeri-m.top/index.php?route=comic/article&c_id=1&comic_id=2",
		BrowserType: "unsupported",
		RuntimeRoot: filepath.Join(workspace, "runtime"),
	}
	req = req.Normalize()
	if req.BrowserType != "firefox" {
		t.Fatalf("BrowserType = %q, want firefox for zeri URL", req.BrowserType)
	}
	wantProfile := filepath.Clean(`runtime/browser-profiles/baseline-userdata`)
	if req.ProfileDir != wantProfile {
		t.Fatalf("ProfileDir = %q, want %q", req.ProfileDir, wantProfile)
	}
	wantBrowserPath := `C:\Program Files\Mozilla Firefox\firefox.exe`
	if req.BrowserPath != wantBrowserPath {
		t.Fatalf("BrowserPath = %q, want %q", req.BrowserPath, wantBrowserPath)
	}
}

func TestNormalizeTaskURLCollapsesRepeatedPrefixes(t *testing.T) {
	raw := "https://zeri-m.top/index.php?https://zeri-m.top/index.php?route=comic/article&c_id=1&comic_id=2"
	want := "https://zeri-m.top/index.php?route=comic/article&c_id=1&comic_id=2"
	if got := NormalizeTaskURL(raw); got != want {
		t.Fatalf("NormalizeTaskURL() = %q, want %q", got, want)
	}
	req := BrowserLaunchRequest{URL: raw}
	if got := req.Normalize().URL; got != want {
		t.Fatalf("Normalize() URL = %q, want %q", got, want)
	}
}

func TestBrowserLaunchRequestUsesFrontendStateDefaults(t *testing.T) {
	workspace := t.TempDir()
	runtimeRoot := filepath.Join(workspace, "runtime")
	browserPath := filepath.Join(workspace, "browsers", "firefox", "firefox.exe")
	installRoot := filepath.Join(workspace, "browsers", "firefox")
	driverDir := filepath.Join(workspace, "browsers", "driver")
	downloadDir := filepath.Join(workspace, "downloads")
	statePath := projectruntime.DefaultFrontendStatePath(runtimeRoot)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("create frontend state dir: %v", err)
	}
	if err := projectruntime.SaveFrontendState(statePath, projectruntime.FrontendState{
		FirefoxExecutablePath: browserPath,
		FirefoxInstallRoot:    installRoot,
		PlaywrightDriverDir:   driverDir,
		DownloadDir:           downloadDir,
	}); err != nil {
		t.Fatalf("save frontend state: %v", err)
	}
	t.Setenv("COMIC_DOWNLOADER_FRONTEND_STATE_PATH", statePath)

	req := BrowserLaunchRequest{
		URL:         "https://example.com",
		RuntimeRoot: runtimeRoot,
	}
	req = req.Normalize()

	if req.BrowserPath != browserPath {
		t.Fatalf("BrowserPath = %q, want frontend state firefox path", req.BrowserPath)
	}
	if req.DriverDir != driverDir {
		t.Fatalf("DriverDir = %q, want frontend state driver dir", req.DriverDir)
	}
	if req.DownloadRoot != downloadDir {
		t.Fatalf("DownloadRoot = %q, want frontend state download dir", req.DownloadRoot)
	}
	if req.OutputDir != downloadDir {
		t.Fatalf("OutputDir = %q, want frontend state download dir", req.OutputDir)
	}
}

func TestBrowserLaunchRequestUsesDownloadDirEnvFallback(t *testing.T) {
	workspace := t.TempDir()
	runtimeRoot := filepath.Join(workspace, "runtime")
	t.Setenv("COMIC_DOWNLOADER_DOWNLOAD_DIR", `D:\portable-data\runtime\output`)

	req := BrowserLaunchRequest{
		URL:         "https://example.com",
		RuntimeRoot: runtimeRoot,
	}
	req = req.Normalize()

	if req.DownloadRoot != `D:\portable-data\runtime\output` {
		t.Fatalf("DownloadRoot = %q, want env fallback", req.DownloadRoot)
	}
	if req.OutputDir != `D:\portable-data\runtime\output` {
		t.Fatalf("OutputDir = %q, want env fallback", req.OutputDir)
	}
}

func TestBrowserLaunchRequestFallsBackFromInvalidOutputRoot(t *testing.T) {
	workspace := t.TempDir()
	runtimeRoot := filepath.Join(workspace, "runtime")
	want := filepath.Join(runtimeRoot, "output")

	req := BrowserLaunchRequest{
		URL:          "https://example.com",
		RuntimeRoot:  runtimeRoot,
		DownloadRoot: `F:\Mangadownload\bad?path`,
		OutputDir:    `F:\Mangadownload\bad?path`,
	}
	req = req.Normalize()

	if req.DownloadRoot != want {
		t.Fatalf("DownloadRoot = %q, want fallback %q", req.DownloadRoot, want)
	}
	if req.OutputDir != want {
		t.Fatalf("OutputDir = %q, want fallback %q", req.OutputDir, want)
	}
}
