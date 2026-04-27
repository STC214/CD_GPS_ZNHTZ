package browser

import "path/filepath"

// BrowserType identifies the browser family used by the middleware.
type BrowserType string

const (
	// BrowserTypeFirefox is the only supported runtime for the first rewrite pass.
	BrowserTypeFirefox BrowserType = "firefox"
)

// ScriptRef describes a runtime init script such as firefox_stealth.js.
type ScriptRef struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// LaunchSpec is the browser-side contract the middleware hands to Playwright.
type LaunchSpec struct {
	BrowserType      BrowserType    `json:"browserType"`
	BrowserPath      string         `json:"browserPath"`
	StealthScript    ScriptRef      `json:"stealthScript"`
	RuntimeRoot      string         `json:"runtimeRoot"`
	URL              string         `json:"url"`
	DriverDir        string         `json:"driverDir"`
	LaunchTimeoutMS  int            `json:"launchTimeoutMs"`
	ProfileDir       string         `json:"profileDir"`
	UserDataDir      string         `json:"userDataDir"`
	UserAgent        string         `json:"userAgent"`
	Locale           string         `json:"locale"`
	TimezoneID       string         `json:"timezoneId"`
	ViewportWidth    int            `json:"viewportWidth"`
	ViewportHeight   int            `json:"viewportHeight"`
	FirefoxUserPrefs map[string]any `json:"firefoxUserPrefs,omitempty"`
	Headless         bool           `json:"headless"`
	Adblock          bool           `json:"adblock"`
}

// Payload is the structured data bundle the middleware passes to the worker layer.
type Payload struct {
	URL              string         `json:"url"`
	DownloadRoot     string         `json:"downloadRoot"`
	OutputDir        string         `json:"outputDir"`
	RuntimeRoot      string         `json:"runtimeRoot"`
	DriverDir        string         `json:"driverDir"`
	LaunchTimeoutMS  int            `json:"launchTimeoutMs"`
	ProfileDir       string         `json:"profileDir"`
	UserAgent        string         `json:"userAgent"`
	Locale           string         `json:"locale"`
	TimezoneID       string         `json:"timezoneId"`
	ViewportWidth    int            `json:"viewportWidth"`
	ViewportHeight   int            `json:"viewportHeight"`
	FirefoxUserPrefs map[string]any `json:"firefoxUserPrefs,omitempty"`
	Headless         bool           `json:"headless"`
	Adblock          bool           `json:"adblock"`
	BrowserType      string         `json:"browserType"`
}

// BrowserMiddlewareData is the stable interface exposed by the middleware layer.
type BrowserMiddlewareData interface {
	URL() string
	BrowserType() BrowserType
	RuntimeRoot() string
	BrowserPath() string
	StealthScript() ScriptRef
	ProfileDir() string
	LaunchData(opts BrowserSessionOptions) LaunchData
	ContextData(opts BrowserSessionOptions) ContextData
	LaunchSpec(opts BrowserSessionOptions) LaunchSpec
	Payload(opts BrowserSessionOptions) Payload
}

// resolveRuntimeScript builds a runtime script reference under runtimeRoot.
func resolveRuntimeScript(runtimeRoot, scriptName string) ScriptRef {
	return ScriptRef{
		Name: scriptName,
		Path: filepath.Join(runtimeRoot, scriptName),
	}
}
