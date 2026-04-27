package tasks

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"comic_downloader_go_playwright_stealth/browser"
	projectruntime "comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/siteflow/hentai2"
	"comic_downloader_go_playwright_stealth/siteflow/hentaiaz"
	"comic_downloader_go_playwright_stealth/siteflow/hitomi"
	"comic_downloader_go_playwright_stealth/siteflow/nyahentai"
	"comic_downloader_go_playwright_stealth/siteflow/zeri"
)

const defaultFirefoxUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:149.0) Gecko/20100101 Firefox/149.0"

var urlSchemePattern = regexp.MustCompile(`(?i)https?://`)

// BrowserLaunchRequest is the task-level browser input that flows into the middleware.
type BrowserLaunchRequest struct {
	URL                  string                      `json:"url"`
	BrowserType          string                      `json:"browserType"`
	Headless             bool                        `json:"headless"`
	LaunchTimeoutMS      int                         `json:"launchTimeoutMs"`
	RuntimeRoot          string                      `json:"runtimeRoot"`
	BrowserPath          string                      `json:"browserPath"`
	BrowserInstallDir    string                      `json:"browserInstallDir"`
	DriverDir            string                      `json:"driverDir"`
	ProfileDir           string                      `json:"profileDir"`
	UserDataDir          string                      `json:"userDataDir"`
	UserAgent            string                      `json:"userAgent"`
	Locale               string                      `json:"locale"`
	TimezoneID           string                      `json:"timezoneId"`
	ViewportWidth        int                         `json:"viewportWidth"`
	ViewportHeight       int                         `json:"viewportHeight"`
	FirefoxUserPrefsJSON string                      `json:"firefoxUserPrefsJson"`
	FirefoxUserPrefs     map[string]any              `json:"firefoxUserPrefs"`
	DownloadRoot         string                      `json:"downloadRoot"`
	OutputDir            string                      `json:"outputDir"`
	Adblock              bool                        `json:"adblock"`
	KeepOpen             bool                        `json:"keepOpen"`
	WorkerID             string                      `json:"workerId"`
	TaskID               string                      `json:"taskId"`
	Progress             func(zeri.DownloadProgress) `json:"-"`
	Context              context.Context             `json:"-"`
}

// LockScope returns a session-lock scope for task-scoped browser sessions.
func (r BrowserLaunchRequest) LockScope() string {
	r = r.Normalize()
	if !r.UsesTaskProfile() {
		return ""
	}
	parts := []string{r.BrowserType}
	if strings.TrimSpace(r.WorkerID) != "" {
		parts = append(parts, r.WorkerID)
	}
	if strings.TrimSpace(r.TaskID) != "" {
		parts = append(parts, r.TaskID)
	}
	return strings.Join(parts, "/")
}

// Normalize returns a cleaned request with defaults applied.
func (r BrowserLaunchRequest) Normalize() BrowserLaunchRequest {
	if strings.TrimSpace(r.RuntimeRoot) == "" {
		r.RuntimeRoot = "runtime"
	}
	if r.LaunchTimeoutMS <= 0 {
		r.LaunchTimeoutMS = 120000
	}
	r.URL = NormalizeTaskURL(r.URL)
	r.BrowserType = normalizeBrowserType(r.BrowserType)
	if zeri.IsZeriURL(r.URL) || hentai2.IsHentai2URL(r.URL) || hentaiaz.IsHentaiazURL(r.URL) || nyahentai.IsNyahentaiURL(r.URL) || hitomi.IsHitomiURL(r.URL) {
		r.BrowserType = string(projectruntime.BrowserTypeFirefox)
	}
	r.BrowserInstallDir = strings.TrimSpace(r.BrowserInstallDir)
	frontendState, hasFrontendState := loadFrontendStateDefaults(r.RuntimeRoot)
	if strings.TrimSpace(r.DriverDir) == "" {
		if hasFrontendState && strings.TrimSpace(frontendState.PlaywrightDriverDir) != "" {
			r.DriverDir = filepath.Clean(frontendState.PlaywrightDriverDir)
		}
	}
	if strings.TrimSpace(r.DriverDir) == "" {
		r.DriverDir = projectruntime.DefaultPlaywrightDriverDir(r.RuntimeRoot)
	}
	selectedProfileDir := strings.TrimSpace(r.ProfileDir)
	userDataDir := strings.TrimSpace(r.UserDataDir)
	r.WorkerID = strings.TrimSpace(r.WorkerID)
	r.TaskID = strings.TrimSpace(r.TaskID)
	r.UserAgent = strings.TrimSpace(r.UserAgent)
	if r.UserAgent == "" {
		r.UserAgent = defaultUserAgentForBrowserType(r.BrowserType)
	}
	if strings.TrimSpace(r.BrowserPath) == "" {
		if hasFrontendState {
			if strings.TrimSpace(frontendState.FirefoxExecutablePath) != "" {
				r.BrowserPath = filepath.Clean(frontendState.FirefoxExecutablePath)
			}
		}
	}
	if strings.TrimSpace(r.BrowserPath) == "" {
		r.BrowserPath = defaultExecutablePathForBrowserType(r.RuntimeRoot, r.BrowserType)
	}
	if strings.TrimSpace(r.BrowserInstallDir) == "" && hasFrontendState {
		if strings.TrimSpace(frontendState.FirefoxInstallRoot) != "" {
			r.BrowserInstallDir = filepath.Clean(frontendState.FirefoxInstallRoot)
		}
	}
	if selectedProfileDir != "" {
		r.ProfileDir = filepath.Clean(selectedProfileDir)
	} else if userDataDir != "" {
		r.ProfileDir = filepath.Clean(userDataDir)
	} else {
		r.ProfileDir = defaultProfileDirForBrowserType(r.BrowserType)
	}
	if userDataDir != "" {
		r.UserDataDir = filepath.Clean(userDataDir)
	} else {
		r.UserDataDir = r.ProfileDir
	}
	safeDefaultOutput := filepath.Join(r.RuntimeRoot, "output")
	r.DownloadRoot = normalizeOutputRoot(
		r.DownloadRoot,
		func() string {
			if hasFrontendState {
				return frontendState.DownloadDir
			}
			return ""
		}(),
		os.Getenv("COMIC_DOWNLOADER_DOWNLOAD_DIR"),
		safeDefaultOutput,
	)
	r.OutputDir = normalizeOutputRoot(
		r.OutputDir,
		func() string {
			if hasFrontendState {
				return frontendState.DownloadDir
			}
			return ""
		}(),
		os.Getenv("COMIC_DOWNLOADER_DOWNLOAD_DIR"),
		safeDefaultOutput,
	)
	if trimmed := strings.TrimSpace(r.FirefoxUserPrefsJSON); trimmed != "" {
		var prefs map[string]any
		if err := json.Unmarshal([]byte(trimmed), &prefs); err == nil {
			r.FirefoxUserPrefs = prefs
		}
	}
	r.Locale = strings.TrimSpace(r.Locale)
	r.TimezoneID = strings.TrimSpace(r.TimezoneID)
	return r
}

// NormalizeTaskURL trims the raw URL and collapses concatenated repeated URL prefixes
// by keeping the last scheme occurrence when multiple http(s):// markers are present.
func NormalizeTaskURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	matches := urlSchemePattern.FindAllStringIndex(raw, -1)
	if len(matches) <= 1 {
		return raw
	}
	last := matches[len(matches)-1][0]
	if last < 0 || last >= len(raw) {
		return raw
	}
	candidate := strings.TrimSpace(raw[last:])
	if candidate == "" {
		return raw
	}
	return candidate
}

func normalizeOutputRoot(primary, frontendDefault, envDefault, safeDefault string) string {
	for _, candidate := range []string{primary, frontendDefault, envDefault, safeDefault} {
		if cleaned, ok := normalizeTaskPathCandidate(candidate); ok {
			return cleaned
		}
	}
	return filepath.Clean(strings.TrimSpace(safeDefault))
}

func normalizeTaskPathCandidate(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	cleaned := filepath.Clean(raw)
	if !isLikelyValidWindowsPath(cleaned) {
		return "", false
	}
	return cleaned, true
}

func isLikelyValidWindowsPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if strings.ContainsAny(path, "<>\"|?*") {
		return false
	}
	for _, r := range path {
		if r < 32 {
			return false
		}
	}
	if idx := strings.IndexByte(path, ':'); idx >= 0 && idx != 1 {
		return false
	}
	return true
}

// UsesTaskProfile reports whether the request should clone the current mother profile into a task temp directory.
func (r BrowserLaunchRequest) UsesTaskProfile() bool {
	return strings.TrimSpace(r.WorkerID) != "" && strings.TrimSpace(r.TaskID) != ""
}

// PrepareTaskProfile copies the current mother profile into a task-scoped temporary profile tree.
func (r BrowserLaunchRequest) PrepareTaskProfile() (projectruntime.BrowserTaskProfile, error) {
	r = r.Normalize()
	manager := projectruntime.NewBrowserProfileManager(workspaceRootFromRuntimeRoot(r.RuntimeRoot))
	return manager.PrepareTaskProfileFromSource(projectruntime.BrowserType(r.BrowserType), r.ProfileDir, r.WorkerID, r.TaskID)
}

// CleanupTaskProfile removes the task-scoped temporary profile tree.
func (r BrowserLaunchRequest) CleanupTaskProfile() error {
	r = r.Normalize()
	manager := projectruntime.NewBrowserProfileManager(workspaceRootFromRuntimeRoot(r.RuntimeRoot))
	return manager.CleanupTaskProfile(projectruntime.BrowserType(r.BrowserType), r.WorkerID, r.TaskID)
}

// BrowserOptions converts the request into browser-layer launch options.
func (r BrowserLaunchRequest) BrowserOptions() browser.BrowserSessionOptions {
	return browser.BrowserSessionOptions{
		URL:               r.URL,
		Headless:          browser.HeadlessPtr(r.Headless),
		LaunchTimeoutMS:   r.LaunchTimeoutMS,
		LockScope:         r.LockScope(),
		AdblockRulesPath:  adblockRulesPathForRuntime(r.RuntimeRoot, r.Adblock),
		DriverDir:         r.DriverDir,
		ProfileDir:        r.ProfileDir,
		BrowserInstallDir: r.BrowserInstallDir,
		UserAgent:         r.UserAgent,
		Locale:            r.Locale,
		TimezoneID:        r.TimezoneID,
		ViewportWidth:     r.ViewportWidth,
		ViewportHeight:    r.ViewportHeight,
		FirefoxUserPrefs:  r.FirefoxUserPrefs,
	}
}

func adblockRulesPathForRuntime(runtimeRoot string, enabled bool) string {
	if !enabled {
		return ""
	}
	runtimeRoot = strings.TrimSpace(runtimeRoot)
	if runtimeRoot == "" {
		runtimeRoot = "runtime"
	}
	return filepath.Join(runtimeRoot, "adblock", "AWAvenue-Ads-Rule.txt")
}

// FirefoxMiddleware builds the browser middleware from the task-level request.
func (r BrowserLaunchRequest) FirefoxMiddleware() browser.FirefoxMiddleware {
	r = r.Normalize()
	middleware := browser.NewFirefoxMiddleware(r.URL).
		WithRuntimeRoot(r.RuntimeRoot).
		WithBrowserPath(r.BrowserPath).
		WithBrowserInstallDir(r.BrowserInstallDir).
		WithDriverDir(r.DriverDir).
		WithLaunchTimeoutMS(r.LaunchTimeoutMS).
		WithProfileDir(r.ProfileDir).
		WithUserDataDir(r.UserDataDir).
		WithUserAgent(r.UserAgent).
		WithLocale(r.Locale).
		WithTimezoneID(r.TimezoneID).
		WithViewport(r.ViewportWidth, r.ViewportHeight).
		WithFirefoxUserPrefs(r.FirefoxUserPrefs).
		WithDownloadRoot(r.DownloadRoot).
		WithOutputDir(r.OutputDir).
		WithHeadless(r.Headless).
		WithAdblock(r.Adblock)
	return middleware
}

func normalizeBrowserType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(projectruntime.BrowserTypeFirefox):
		return string(projectruntime.BrowserTypeFirefox)
	default:
		return string(projectruntime.BrowserTypeFirefox)
	}
}

func defaultProfileDirForBrowserType(browserType string) string {
	return projectruntime.DefaultFirefoxProfileDir()
}

func defaultExecutablePathForBrowserType(runtimeRoot, browserType string) string {
	return projectruntime.DefaultFirefoxExecutablePath(runtimeRoot)
}

func defaultUserAgentForBrowserType(browserType string) string {
	return defaultFirefoxUserAgent
}

func workspaceRootFromRuntimeRoot(runtimeRoot string) string {
	runtimeRoot = filepath.Clean(strings.TrimSpace(runtimeRoot))
	if runtimeRoot == "" || runtimeRoot == "." {
		return "."
	}
	return filepath.Dir(runtimeRoot)
}

func loadFrontendStateDefaults(runtimeRoot string) (projectruntime.FrontendState, bool) {
	workspaceRoot := workspaceRootFromRuntimeRoot(runtimeRoot)
	statePath := projectruntime.ResolveFrontendStatePath(workspaceRoot)
	state, err := projectruntime.LoadFrontendState(statePath)
	if err != nil {
		return projectruntime.FrontendState{}, false
	}
	if strings.TrimSpace(state.DownloadDir) != "" {
		state.DownloadDir = projectruntime.ResolvePath(workspaceRoot, state.DownloadDir)
	}
	return state, true
}
