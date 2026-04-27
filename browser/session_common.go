package browser

import "strings"

// LaunchData is the Playwright launch input distilled into a dependency-free shape.
type LaunchData struct {
	ExecutablePath string `json:"executablePath"`
	Headless       bool   `json:"headless"`
}

// ContextData is the Playwright context input distilled into a dependency-free shape.
type ContextData struct {
	BaseURL string `json:"baseURL"`
}

// FirefoxSession is the runtime handle returned by Open().
type FirefoxSession struct {
	Middleware  FirefoxMiddleware
	URL         string
	Playwright  any
	Browser     any
	Context     any
	Page        any
	releaseLock func() error
	closed      chan struct{}
}

// BrowserPageActions is the minimal page operation surface used by site flows.
type BrowserPageActions interface {
	PageURL() string
	Content() (string, error)
	Goto(url string) error
	ClickText(text string) error
	LoadLazyContent() error
	LoadLazyContentForCount(expectedImageCount int) error
}

// LaunchData returns the launch inputs needed by the Playwright-backed implementation.
func (m FirefoxMiddleware) LaunchData(opts BrowserSessionOptions) LaunchData {
	spec := m.LaunchSpec(opts)
	return LaunchData{
		ExecutablePath: spec.BrowserPath,
		Headless:       m.resolveHeadless(opts),
	}
}

// ContextData returns the browser context inputs needed by the Playwright-backed implementation.
func (m FirefoxMiddleware) ContextData(opts BrowserSessionOptions) ContextData {
	return ContextData{BaseURL: m.resolveURL(opts)}
}

// Open delegates to the build-specific implementation helper.
func (m FirefoxMiddleware) Open(opts BrowserSessionOptions) (*FirefoxSession, error) {
	return openFirefoxSession(m, opts)
}

// Close delegates to the build-specific implementation helper.
func (s *FirefoxSession) Close() error {
	return closeFirefoxSession(s)
}

// Title returns the page title from the live browser session.
func (s *FirefoxSession) Title() (string, error) {
	return sessionTitle(s)
}

// WaitClosed blocks until the browser page is closed by the user or browser runtime.
func (s *FirefoxSession) WaitClosed() error {
	return waitFirefoxSessionClosed(s)
}

// PageURL returns the current page URL for the live browser session.
func (s *FirefoxSession) PageURL() string {
	if s == nil {
		return ""
	}
	return s.URL
}

// Content returns the current page HTML for the live browser session.
func (s *FirefoxSession) Content() (string, error) {
	return sessionContent(s)
}

// Goto navigates the live browser session to the provided URL.
func (s *FirefoxSession) Goto(url string) error {
	return sessionGoto(s, url)
}

// ClickText clicks a visible text node in the live browser session.
func (s *FirefoxSession) ClickText(text string) error {
	return sessionClickText(s, text)
}

// LoadLazyContent scrolls through the page to trigger lazy-loaded content.
func (s *FirefoxSession) LoadLazyContent() error {
	return sessionLoadLazyContentForCount(s, 0)
}

// LoadLazyContentForCount scrolls through the page until the expected image count is loaded.
func (s *FirefoxSession) LoadLazyContentForCount(expectedImageCount int) error {
	return sessionLoadLazyContentForCount(s, expectedImageCount)
}

func normalizedURL(value string) string {
	return strings.TrimSpace(value)
}
