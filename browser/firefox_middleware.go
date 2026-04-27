package browser

import (
	"path/filepath"
	"strings"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
)

const defaultFirefoxUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:149.0) Gecko/20100101 Firefox/149.0"

// FirefoxMiddleware carries the browser-side state for one URL.
type FirefoxMiddleware struct {
	url               string
	runtimeRoot       string
	downloadRoot      string
	outputDir         string
	driverDir         string
	launchTimeoutMS   int
	browserInstallDir string
	profileDir        string
	userDataDir       string
	userAgent         string
	locale            string
	timezoneID        string
	viewportWidth     int
	viewportHeight    int
	firefoxUserPrefs  map[string]any
	browserPath       string
	headless          bool
	adblock           bool
}

// NewFirefoxMiddleware builds a Firefox-only browser middleware state object.
func NewFirefoxMiddleware(url string) FirefoxMiddleware {
	return FirefoxMiddleware{
		url:               strings.TrimSpace(url),
		runtimeRoot:       "runtime",
		downloadRoot:      "",
		outputDir:         "",
		driverDir:         "",
		launchTimeoutMS:   120000,
		browserInstallDir: "",
		profileDir:        projectruntime.DefaultFirefoxProfileDir(),
		userDataDir:       "",
		userAgent:         defaultFirefoxUserAgent,
		locale:            "",
		timezoneID:        "",
		viewportWidth:     1365,
		viewportHeight:    768,
		firefoxUserPrefs:  nil,
		browserPath:       projectruntime.DefaultFirefoxExecutablePath("runtime"),
		headless:          true,
		adblock:           true,
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// WithRuntimeRoot sets the runtime root used to resolve firefox_stealth.js.
func (m FirefoxMiddleware) WithRuntimeRoot(runtimeRoot string) FirefoxMiddleware {
	m.runtimeRoot = normalizePath(runtimeRoot)
	if m.runtimeRoot == "" {
		m.runtimeRoot = "runtime"
	}
	if m.browserPath == "" {
		m.browserPath = projectruntime.DefaultFirefoxExecutablePath(m.RuntimeRoot())
	}
	return m
}

// WithDownloadRoot sets the output/download root used by the worker layer.
func (m FirefoxMiddleware) WithDownloadRoot(downloadRoot string) FirefoxMiddleware {
	m.downloadRoot = normalizePath(downloadRoot)
	return m
}

// WithOutputDir sets the resolved output directory.
func (m FirefoxMiddleware) WithOutputDir(outputDir string) FirefoxMiddleware {
	m.outputDir = normalizePath(outputDir)
	return m
}

// WithDriverDir sets the Playwright driver directory.
func (m FirefoxMiddleware) WithDriverDir(driverDir string) FirefoxMiddleware {
	m.driverDir = normalizePath(driverDir)
	return m
}

// WithLaunchTimeoutMS sets the maximum time to wait for Playwright to start the browser.
func (m FirefoxMiddleware) WithLaunchTimeoutMS(launchTimeoutMS int) FirefoxMiddleware {
	if launchTimeoutMS >= 0 {
		m.launchTimeoutMS = launchTimeoutMS
	}
	return m
}

// WithBrowserInstallDir sets the browser install directory used to resolve the driver when needed.
func (m FirefoxMiddleware) WithBrowserInstallDir(browserInstallDir string) FirefoxMiddleware {
	m.browserInstallDir = normalizePath(browserInstallDir)
	return m
}

// WithUserAgent sets the browser user agent for the launched context.
func (m FirefoxMiddleware) WithUserAgent(userAgent string) FirefoxMiddleware {
	m.userAgent = strings.TrimSpace(userAgent)
	return m
}

// WithProfileDir sets the selected Firefox profile directory.
func (m FirefoxMiddleware) WithProfileDir(profileDir string) FirefoxMiddleware {
	m.profileDir = normalizePath(profileDir)
	return m
}

// WithUserDataDir sets the Firefox profile directory.
func (m FirefoxMiddleware) WithUserDataDir(userDataDir string) FirefoxMiddleware {
	m.userDataDir = normalizePath(userDataDir)
	return m
}

// WithLocale sets the browser locale for a stable testing environment.
func (m FirefoxMiddleware) WithLocale(locale string) FirefoxMiddleware {
	m.locale = strings.TrimSpace(locale)
	return m
}

// WithTimezoneID sets the browser timezone for a stable testing environment.
func (m FirefoxMiddleware) WithTimezoneID(timezoneID string) FirefoxMiddleware {
	m.timezoneID = strings.TrimSpace(timezoneID)
	return m
}

// WithViewport sets the browser viewport for a stable testing environment.
func (m FirefoxMiddleware) WithViewport(width, height int) FirefoxMiddleware {
	if width > 0 {
		m.viewportWidth = width
	}
	if height > 0 {
		m.viewportHeight = height
	}
	return m
}

// WithFirefoxUserPrefs sets Firefox user preferences for the launch process.
func (m FirefoxMiddleware) WithFirefoxUserPrefs(userPrefs map[string]any) FirefoxMiddleware {
	if len(userPrefs) == 0 {
		m.firefoxUserPrefs = nil
		return m
	}
	m.firefoxUserPrefs = userPrefs
	return m
}

// WithBrowserPath sets the Firefox executable path.
func (m FirefoxMiddleware) WithBrowserPath(browserPath string) FirefoxMiddleware {
	m.browserPath = normalizePath(browserPath)
	return m
}

// WithHeadless switches the launch mode.
func (m FirefoxMiddleware) WithHeadless(headless bool) FirefoxMiddleware {
	m.headless = headless
	return m
}

// WithAdblock switches adblock injection on or off.
func (m FirefoxMiddleware) WithAdblock(adblock bool) FirefoxMiddleware {
	m.adblock = adblock
	return m
}

func (m FirefoxMiddleware) URL() string {
	return m.url
}

func (m FirefoxMiddleware) BrowserType() BrowserType {
	return BrowserTypeFirefox
}

func (m FirefoxMiddleware) RuntimeRoot() string {
	if strings.TrimSpace(m.runtimeRoot) == "" {
		return "runtime"
	}
	return m.runtimeRoot
}

func (m FirefoxMiddleware) BrowserPath() string {
	if strings.TrimSpace(m.browserPath) == "" {
		return projectruntime.DefaultFirefoxExecutablePath(m.RuntimeRoot())
	}
	return m.browserPath
}

func (m FirefoxMiddleware) StealthScript() ScriptRef {
	return resolveRuntimeScript(m.RuntimeRoot(), "firefox_stealth.js")
}

func (m FirefoxMiddleware) ProfileDir() string {
	if strings.TrimSpace(m.profileDir) == "" {
		return projectruntime.DefaultFirefoxProfileDir()
	}
	return m.profileDir
}

func (m FirefoxMiddleware) resolveProfileDir(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(opts.ProfileDir); trimmed != "" {
		return filepath.Clean(trimmed)
	}
	return m.ProfileDir()
}

func (m FirefoxMiddleware) resolveURL(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(opts.URL); trimmed != "" {
		return trimmed
	}
	return m.URL()
}

func (m FirefoxMiddleware) resolveHeadless(opts BrowserSessionOptions) bool {
	if opts.Headless != nil {
		return *opts.Headless
	}
	return m.headless
}

func (m FirefoxMiddleware) resolveUserDataDir(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(m.userDataDir); trimmed != "" {
		return filepath.Clean(trimmed)
	}
	return m.resolveProfileDir(opts)
}

func (m FirefoxMiddleware) resolveDriverDir(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(opts.DriverDir); trimmed != "" {
		return filepath.Clean(trimmed)
	}
	return m.driverDir
}

func (m FirefoxMiddleware) resolveDriverDirOrDefault(opts BrowserSessionOptions) string {
	if resolved := m.resolveDriverDir(opts); strings.TrimSpace(resolved) != "" {
		return resolved
	}
	if strings.TrimSpace(m.browserInstallDir) != "" {
		return filepath.Join(m.browserInstallDir, "driver")
	}
	return projectruntime.DefaultPlaywrightDriverDir(m.RuntimeRoot())
}

func (m FirefoxMiddleware) resolveLaunchTimeoutMS(opts BrowserSessionOptions) int {
	if opts.LaunchTimeoutMS >= 0 {
		return opts.LaunchTimeoutMS
	}
	if m.launchTimeoutMS >= 0 {
		return m.launchTimeoutMS
	}
	return 120000
}

func (m FirefoxMiddleware) resolveLocale(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(opts.Locale); trimmed != "" {
		return trimmed
	}
	return m.locale
}

func (m FirefoxMiddleware) resolveUserAgent(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(opts.UserAgent); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(m.userAgent)
}

func (m FirefoxMiddleware) resolveTimezoneID(opts BrowserSessionOptions) string {
	if trimmed := strings.TrimSpace(opts.TimezoneID); trimmed != "" {
		return trimmed
	}
	return m.timezoneID
}

func (m FirefoxMiddleware) resolveFirefoxUserPrefs(opts BrowserSessionOptions) map[string]any {
	if len(opts.FirefoxUserPrefs) > 0 {
		return opts.FirefoxUserPrefs
	}
	return m.firefoxUserPrefs
}

func (m FirefoxMiddleware) resolveViewport(opts BrowserSessionOptions) (int, int) {
	width := m.viewportWidth
	height := m.viewportHeight
	if opts.ViewportWidth > 0 {
		width = opts.ViewportWidth
	}
	if opts.ViewportHeight > 0 {
		height = opts.ViewportHeight
	}
	return width, height
}

func (m FirefoxMiddleware) LaunchSpec(opts BrowserSessionOptions) LaunchSpec {
	width, height := m.resolveViewport(opts)
	return LaunchSpec{
		BrowserType:      m.BrowserType(),
		BrowserPath:      m.BrowserPath(),
		StealthScript:    m.StealthScript(),
		RuntimeRoot:      m.RuntimeRoot(),
		URL:              m.resolveURL(opts),
		ProfileDir:       m.resolveProfileDir(opts),
		UserDataDir:      m.resolveUserDataDir(opts),
		UserAgent:        m.resolveUserAgent(opts),
		Locale:           m.resolveLocale(opts),
		TimezoneID:       m.resolveTimezoneID(opts),
		ViewportWidth:    width,
		ViewportHeight:   height,
		FirefoxUserPrefs: m.resolveFirefoxUserPrefs(opts),
		Headless:         m.resolveHeadless(opts),
		Adblock:          m.adblock,
		DriverDir:        m.resolveDriverDirOrDefault(opts),
		LaunchTimeoutMS:  m.resolveLaunchTimeoutMS(opts),
	}
}

func (m FirefoxMiddleware) Payload(opts BrowserSessionOptions) Payload {
	width, height := m.resolveViewport(opts)
	return Payload{
		URL:              m.resolveURL(opts),
		DownloadRoot:     m.downloadRoot,
		OutputDir:        m.outputDir,
		RuntimeRoot:      m.RuntimeRoot(),
		ProfileDir:       m.resolveProfileDir(opts),
		UserAgent:        m.resolveUserAgent(opts),
		Locale:           m.resolveLocale(opts),
		TimezoneID:       m.resolveTimezoneID(opts),
		ViewportWidth:    width,
		ViewportHeight:   height,
		FirefoxUserPrefs: m.resolveFirefoxUserPrefs(opts),
		Headless:         m.resolveHeadless(opts),
		Adblock:          m.adblock,
		BrowserType:      string(m.BrowserType()),
		DriverDir:        m.resolveDriverDirOrDefault(opts),
	}
}
