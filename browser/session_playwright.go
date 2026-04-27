//go:build playwright

package browser

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/playwright-community/playwright-go"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
)

func (m FirefoxMiddleware) toPlaywrightLaunchOptions(opts BrowserSessionOptions) playwright.BrowserTypeLaunchOptions {
	data := m.LaunchData(opts)
	return playwright.BrowserTypeLaunchOptions{
		ExecutablePath:   playwright.String(data.ExecutablePath),
		Headless:         playwright.Bool(data.Headless),
		FirefoxUserPrefs: m.resolveFirefoxUserPrefs(opts),
	}
}

func (m FirefoxMiddleware) toPlaywrightContextOptions(opts BrowserSessionOptions) playwright.BrowserNewContextOptions {
	data := m.ContextData(opts)
	contextOptions := playwright.BrowserNewContextOptions{}
	if strings.TrimSpace(data.BaseURL) != "" {
		contextOptions.BaseURL = playwright.String(data.BaseURL)
	}
	if userAgent := strings.TrimSpace(m.resolveUserAgent(opts)); userAgent != "" {
		contextOptions.UserAgent = playwright.String(userAgent)
	}
	if locale := strings.TrimSpace(m.resolveLocale(opts)); locale != "" {
		contextOptions.Locale = playwright.String(locale)
	}
	if timezoneID := strings.TrimSpace(m.resolveTimezoneID(opts)); timezoneID != "" {
		contextOptions.TimezoneId = playwright.String(timezoneID)
	}
	if width, height := m.resolveViewport(opts); width > 0 && height > 0 {
		contextOptions.Viewport = &playwright.Size{Width: width, Height: height}
	}
	return contextOptions
}

func (m FirefoxMiddleware) toPlaywrightPersistentContextOptions(opts BrowserSessionOptions) playwright.BrowserTypeLaunchPersistentContextOptions {
	contextOptions := playwright.BrowserTypeLaunchPersistentContextOptions{
		ExecutablePath:   playwright.String(m.BrowserPath()),
		Headless:         playwright.Bool(m.resolveHeadless(opts)),
		FirefoxUserPrefs: m.resolveFirefoxUserPrefs(opts),
		Timeout:          playwright.Float(float64(m.resolveLaunchTimeoutMS(opts))),
	}
	if userAgent := strings.TrimSpace(m.resolveUserAgent(opts)); userAgent != "" {
		contextOptions.UserAgent = playwright.String(userAgent)
	}
	if locale := strings.TrimSpace(m.resolveLocale(opts)); locale != "" {
		contextOptions.Locale = playwright.String(locale)
	}
	if timezoneID := strings.TrimSpace(m.resolveTimezoneID(opts)); timezoneID != "" {
		contextOptions.TimezoneId = playwright.String(timezoneID)
	}
	if width, height := m.resolveViewport(opts); width > 0 && height > 0 {
		contextOptions.Viewport = &playwright.Size{Width: width, Height: height}
	}
	return contextOptions
}

func openFirefoxSession(m FirefoxMiddleware, opts BrowserSessionOptions) (*FirefoxSession, error) {
	spec := m.LaunchSpec(opts)
	if strings.TrimSpace(spec.URL) == "" {
		return nil, errors.New("browser middleware url is empty")
	}
	if strings.TrimSpace(spec.BrowserPath) == "" {
		return nil, errors.New("browser path is empty")
	}
	if _, err := os.Stat(spec.BrowserPath); err != nil {
		return nil, fmt.Errorf("browser executable %q: %w", spec.BrowserPath, err)
	}
	if _, err := os.Stat(spec.StealthScript.Path); err != nil {
		return nil, fmt.Errorf("stealth script %q: %w", spec.StealthScript.Path, err)
	}

	previousDriverPath := os.Getenv("PLAYWRIGHT_DRIVER_PATH")
	if driverDir := strings.TrimSpace(spec.DriverDir); driverDir != "" {
		if err := os.Setenv("PLAYWRIGHT_DRIVER_PATH", driverDir); err != nil {
			return nil, fmt.Errorf("set PLAYWRIGHT_DRIVER_PATH: %w", err)
		}
		defer func() {
			if previousDriverPath == "" {
				_ = os.Unsetenv("PLAYWRIGHT_DRIVER_PATH")
				return
			}
			_ = os.Setenv("PLAYWRIGHT_DRIVER_PATH", previousDriverPath)
		}()
	}

	releaseLock, err := projectruntime.AcquireBrowserSessionLockScoped(m.RuntimeRoot(), opts.LockScope)
	if err != nil {
		return nil, err
	}

	pw, err := playwright.Run()
	if err != nil {
		_ = releaseLock()
		return nil, fmt.Errorf("start playwright: %w", err)
	}

	persistentOptions := m.toPlaywrightPersistentContextOptions(opts)
	context, err := pw.Firefox.LaunchPersistentContext(spec.UserDataDir, persistentOptions)
	if err != nil {
		_ = pw.Stop()
		_ = releaseLock()
		return nil, fmt.Errorf("launch firefox: %w", err)
	}

	if err := context.AddInitScript(playwright.Script{
		Path: playwright.String(spec.StealthScript.Path),
	}); err != nil {
		_ = context.Close()
		_ = pw.Stop()
		_ = releaseLock()
		return nil, fmt.Errorf("add stealth init script: %w", err)
	}
	if err := applyAdblockRules(context, opts.AdblockRulesPath); err != nil {
		_ = context.Close()
		_ = pw.Stop()
		_ = releaseLock()
		return nil, fmt.Errorf("apply adblock rules: %w", err)
	}

	var page playwright.Page
	pages := context.Pages()
	fmt.Printf("browser session pages before goto: %d\n", len(pages))
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = context.NewPage()
	}
	if err != nil {
		_ = context.Close()
		_ = pw.Stop()
		_ = releaseLock()
		return nil, fmt.Errorf("create firefox page: %w", err)
	}
	fmt.Printf("browser session selected page before goto: %s\n", page.URL())

	closed := make(chan struct{})
	var closedOnce sync.Once
	page.OnClose(func(playwright.Page) {
		closedOnce.Do(func() {
			close(closed)
		})
	})

	targetURL := spec.URL
	if strings.TrimSpace(targetURL) == "" {
		targetURL = m.URL()
	}
	if _, err := page.Goto(targetURL); err != nil {
		_ = page.Close()
		_ = context.Close()
		_ = pw.Stop()
		_ = releaseLock()
		return nil, fmt.Errorf("goto %q: %w", targetURL, err)
	}
	fmt.Printf("browser session selected page after goto: %s\n", page.URL())
	return &FirefoxSession{
		Middleware:  m,
		URL:         targetURL,
		Playwright:  pw,
		Browser:     nil,
		Context:     context,
		Page:        page,
		releaseLock: releaseLock,
		closed:      closed,
	}, nil
}

func closeFirefoxSession(s *FirefoxSession) error {
	if s == nil {
		return nil
	}
	var firstErr error
	if page, ok := s.Page.(playwright.Page); ok && page != nil {
		if err := page.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if context, ok := s.Context.(playwright.BrowserContext); ok && context != nil {
		if err := context.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if browser, ok := s.Browser.(playwright.Browser); ok && browser != nil {
		if err := browser.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if pw, ok := s.Playwright.(*playwright.Playwright); ok && pw != nil {
		if err := pw.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.releaseLock != nil {
		if err := s.releaseLock(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func sessionTitle(s *FirefoxSession) (string, error) {
	if s == nil {
		return "", errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return "", errors.New("browser session page is not a playwright.Page")
	}
	return page.Title()
}

func waitFirefoxSessionClosed(s *FirefoxSession) error {
	if s == nil {
		return errors.New("browser session is nil")
	}
	if s.closed == nil {
		return errors.New("browser session close channel is nil")
	}
	<-s.closed
	return nil
}

func sessionContent(s *FirefoxSession) (string, error) {
	if s == nil {
		return "", errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return "", errors.New("browser session page is not a playwright.Page")
	}
	return page.Content()
}

func sessionGoto(s *FirefoxSession, url string) error {
	if s == nil {
		return errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return errors.New("browser session page is not a playwright.Page")
	}
	if _, err := page.Goto(url); err != nil {
		return err
	}
	s.URL = url
	return nil
}

func sessionClickText(s *FirefoxSession, text string) error {
	if s == nil {
		return errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return errors.New("browser session page is not a playwright.Page")
	}
	if strings.TrimSpace(text) == "100%" {
		button := page.Locator("#image_width1 button")
		if count, err := button.Count(); err == nil && count > 0 {
			box, err := button.First().BoundingBox()
			if err == nil && box != nil {
				return page.Mouse().Click(box.X+box.Width/2, box.Y+box.Height/2)
			}
			return button.First().Click()
		}
	}
	locator := page.GetByText(text, playwright.PageGetByTextOptions{Exact: playwright.Bool(true)})
	return locator.Click()
}

func sessionLoadLazyContentForCount(s *FirefoxSession, expectedImageCount int) error {
	if s == nil {
		return errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return errors.New("browser session page is not a playwright.Page")
	}
	result, err := page.Evaluate(fmt.Sprintf(`async () => {
		const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
		const expected = Math.max(0, Number(%d || 0));
		const imageStats = () => {
			const images = Array.from(document.images || []);
			const total = images.length;
			const loaded = images.filter(img => img.complete && img.naturalWidth > 0).length;
			const target = expected > 0 ? expected : total;
			return { total, loaded, target, allLoaded: total > 0 && loaded === total };
		};
		const scrollTop = () => window.scrollTo(0, 0);
		const scrollBottom = () => window.scrollTo(0, Math.max(0, document.documentElement.scrollHeight - window.innerHeight));
		const scrollBounce = async () => {
			const maxScroll = Math.max(0, document.documentElement.scrollHeight - window.innerHeight);
			const points = [
				0,
				maxScroll * 0.25,
				maxScroll * 0.5,
				maxScroll * 0.75,
				maxScroll,
				maxScroll * 0.75,
				maxScroll * 0.5,
				maxScroll * 0.25,
			];
			for (const point of points) {
				window.scrollTo(0, Math.max(0, Math.floor(point)));
				window.dispatchEvent(new Event('scroll'));
				await sleep(180);
			}
		};
		for (let i = 0; i < 60; i++) {
			const stats = imageStats();
			if (expected > 0 && stats.total >= stats.target && stats.loaded >= stats.target) {
				scrollTop();
				await sleep(150);
				return stats;
			}
			if (expected <= 0 && stats.allLoaded) {
				scrollTop();
				await sleep(150);
				return stats;
			}
			await scrollBounce();
			scrollBottom();
			window.dispatchEvent(new Event('scroll'));
			await sleep(180);
		}
		const stats = imageStats();
		scrollTop();
		await sleep(150);
		return stats;
	}`, expectedImageCount))
	if err != nil {
		return err
	}
	if stats, ok := result.(map[string]any); ok {
		total := int(asFloat64(stats["total"]))
		loaded := int(asFloat64(stats["loaded"]))
		target := int(asFloat64(stats["target"]))
		if target <= 0 {
			target = total
		}
		if target > 0 && loaded < target {
			return fmt.Errorf("lazy images timed out: %d/%d loaded", loaded, target)
		}
	}
	return nil
}

func asFloat64(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
