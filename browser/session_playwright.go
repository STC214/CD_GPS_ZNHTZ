//go:build playwright

package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"

	"comic_downloader_go_playwright_stealth/netproxy"
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
	proxyServer, err := netproxy.NormalizeServer(m.resolveProxyServer(opts))
	if err != nil {
		proxyServer = ""
	}
	contextOptions := playwright.BrowserTypeLaunchPersistentContextOptions{
		ExecutablePath:   playwright.String(m.BrowserPath()),
		Headless:         playwright.Bool(m.resolveHeadless(opts)),
		FirefoxUserPrefs: m.resolveFirefoxUserPrefs(opts),
		Timeout:          playwright.Float(float64(m.resolveLaunchTimeoutMS(opts))),
	}
	if proxyServer != "" {
		contextOptions.Proxy = &playwright.Proxy{Server: proxyServer}
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
	if _, err := netproxy.NormalizeServer(spec.ProxyServer); err != nil {
		return nil, err
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
	if err := gotoWithRetry(page, targetURL); err != nil {
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
	if err := gotoWithRetry(page, url); err != nil {
		return err
	}
	s.URL = url
	return nil
}

func gotoWithRetry(page playwright.Page, url string) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := page.Goto(url); err != nil {
			lastErr = err
			if !isRetryableGotoError(err) || attempt == 3 {
				break
			}
			fmt.Printf("browser goto retry %d/3 for %q after: %v\n", attempt+1, url, err)
			time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
			continue
		}
		return nil
	}
	return lastErr
}

func isRetryableGotoError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, token := range []string{
		"ns_error_net_interrupt",
		"ns_error_net_reset",
		"net::err",
		"navigation failed because page was closed",
	} {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
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

func sessionLoadLazyContentInSelector(s *FirefoxSession, selector string) error {
	if s == nil {
		return errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return errors.New("browser session page is not a playwright.Page")
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return errors.New("lazy selector is empty")
	}
	result, err := page.Evaluate(`async (selector) => {
		const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
		const minSize = 300;
		const startedAt = Date.now();
		const signatureFromLocation = () => {
			const value = String(window.location.pathname || window.location.href || '');
			const reMatch = value.match(/\/fanzine\/re(\d{6,})/i) || value.match(/re(\d{6,})/i);
			if (reMatch) return reMatch[1];
			const numericMatch = value.match(/\d{6,}/);
			return numericMatch ? numericMatch[0] : '';
		};
		const imageURL = (img) => {
			const candidates = [
				img.currentSrc,
				img.src,
				img.getAttribute('data-src'),
				img.getAttribute('data-original'),
				img.getAttribute('data-lazy-src'),
				img.getAttribute('data-url'),
				img.getAttribute('data-srcset'),
				img.getAttribute('srcset'),
			];
			for (const raw of candidates) {
				if (!raw) continue;
				const first = String(raw).split(',')[0].trim().split(/\s+/)[0];
				if (first) return first;
			}
			return '';
		};
		const numericAttr = (img, name) => {
			const value = parseInt(img.getAttribute(name) || '0', 10);
			return Number.isFinite(value) ? value : 0;
		};
		const imageStats = () => {
			const root = document.querySelector(selector);
			if (!root) {
				return { exists: false, total: 0, loaded: 0, targetTotal: 0, targetSized: 0, complete: false, sizedComplete: false };
			}
			const signature = signatureFromLocation();
			const images = Array.from(root.querySelectorAll('img'));
			const targetImages = signature ? images.filter(img => imageURL(img).includes(signature)) : images;
			const total = images.length;
			const loaded = images.filter(img => img.complete && img.naturalWidth > 0).length;
			const targetTotal = targetImages.length;
			const targetSized = targetImages.filter(img => numericAttr(img, 'width') >= minSize && numericAttr(img, 'height') >= minSize).length;
			return {
				exists: true,
				total,
				loaded,
				targetTotal,
				targetSized,
				signature,
				elapsedMS: Date.now() - startedAt,
				complete: total > 0 && loaded === total,
				sizedComplete: targetTotal > 0 && targetSized === targetTotal,
			};
		};
		const scrollTop = () => window.scrollTo(0, 0);
		const scrollToElement = () => {
			const root = document.querySelector(selector);
			if (root) root.scrollIntoView({ block: 'start' });
		};
		const scrollBounce = async () => {
			const root = document.querySelector(selector);
			const docMax = Math.max(0, document.documentElement.scrollHeight - window.innerHeight);
			const rect = root ? root.getBoundingClientRect() : null;
			const rootTop = root ? Math.max(0, window.scrollY + rect.top - 80) : 0;
			const rootBottom = root ? Math.max(rootTop, window.scrollY + rect.bottom - window.innerHeight + 80) : docMax;
			const step = Math.max(240, Math.floor(window.innerHeight * 0.72));
			const points = [];
			for (let point = rootTop; point < rootBottom; point += step) {
				points.push(point);
			}
			points.push(rootBottom, docMax);
			for (const point of points) {
				window.scrollTo(0, Math.max(0, Math.floor(point)));
				window.dispatchEvent(new Event('scroll'));
				await sleep(70);
			}
		};
		let stable = 0;
		let previousTotal = -1;
		let previousLoaded = -1;
		let lastSizeCheck = 0;
		for (let i = 0; i < 90; i++) {
			scrollToElement();
			await sleep(40);
			await scrollBounce();
			const stats = imageStats();
			if (!stats.exists) {
				await sleep(120);
				continue;
			}
			if (stats.elapsedMS - lastSizeCheck >= 5000) {
				lastSizeCheck = stats.elapsedMS;
				if (stats.sizedComplete) {
					scrollTop();
					await sleep(100);
					return stats;
				}
			}
			if (stats.complete && stats.total === previousTotal && stats.loaded === previousLoaded) {
				stable++;
			} else {
				stable = 0;
			}
			previousTotal = stats.total;
			previousLoaded = stats.loaded;
			if (stats.complete && stable >= 3) {
				scrollTop();
				await sleep(100);
				return stats;
			}
			await sleep(120);
		}
		const stats = imageStats();
		scrollTop();
		await sleep(100);
		return stats;
	}`, selector)
	if err != nil {
		return err
	}
	if stats, ok := result.(map[string]any); ok {
		exists, _ := stats["exists"].(bool)
		total := int(asFloat64(stats["total"]))
		loaded := int(asFloat64(stats["loaded"]))
		targetTotal := int(asFloat64(stats["targetTotal"]))
		targetSized := int(asFloat64(stats["targetSized"]))
		if !exists {
			return fmt.Errorf("lazy selector %q not found", selector)
		}
		if total <= 0 {
			return fmt.Errorf("lazy selector %q has no images", selector)
		}
		if targetTotal > 0 && targetSized >= targetTotal {
			return nil
		}
		if loaded < total {
			return fmt.Errorf("lazy selector %q images timed out: %d/%d loaded, sized targets %d/%d", selector, loaded, total, targetSized, targetTotal)
		}
	}
	return nil
}

func sessionImageRecords(s *FirefoxSession) ([]PageImageRecord, error) {
	if s == nil {
		return nil, errors.New("browser session is nil")
	}
	page, ok := s.Page.(playwright.Page)
	if !ok || page == nil {
		return nil, errors.New("browser session page is not a playwright.Page")
	}
	result, err := page.Evaluate(`() => JSON.stringify(Array.from(document.images || []).map((img) => ({
		src: img.currentSrc || img.src || img.getAttribute('data-src') || img.getAttribute('data-original') || img.getAttribute('data-lazy-src') || img.getAttribute('data-url') || '',
		attrWidth: parseInt(img.getAttribute('width') || '0', 10) || 0,
		attrHeight: parseInt(img.getAttribute('height') || '0', 10) || 0,
		naturalWidth: img.naturalWidth || 0,
		naturalHeight: img.naturalHeight || 0,
		offsetWidth: img.offsetWidth || 0,
		offsetHeight: img.offsetHeight || 0,
		clientWidth: img.clientWidth || 0,
		clientHeight: img.clientHeight || 0,
		rectWidth: Math.round(img.getBoundingClientRect().width || 0),
		rectHeight: Math.round(img.getBoundingClientRect().height || 0),
		complete: !!img.complete,
		alt: img.alt || '',
		className: img.className || '',
		id: img.id || '',
		loading: img.loading || ''
	})))`)
	if err != nil {
		return nil, err
	}
	raw, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("image records returned %T", result)
	}
	var records []PageImageRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, fmt.Errorf("decode image records: %w", err)
	}
	return records, nil
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
