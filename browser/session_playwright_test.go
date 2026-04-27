//go:build playwright

package browser

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestFirefoxMiddlewareOpenLocalServer(t *testing.T) {
	if os.Getenv("RUN_PLAYWRIGHT_BROWSER_TEST") != "1" {
		t.Skip("set RUN_PLAYWRIGHT_BROWSER_TEST=1 to run the real browser probe")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Browser Probe</title></head><body>ok</body></html>`))
	}))
	defer server.Close()

	workspaceRoot := filepath.Clean(filepath.FromSlash("F:/Project/comic_downloader_GO_Playwright_stealth"))
	middleware := NewFirefoxMiddleware(server.URL).
		WithRuntimeRoot(filepath.Join(workspaceRoot, "runtime")).
		WithBrowserPath(`C:\Program Files\Mozilla Firefox\firefox.exe`).
		WithHeadless(true)

	session, err := middleware.Open(BrowserSessionOptions{Headless: HeadlessPtr(true)})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	page, ok := session.Page.(playwright.Page)
	if !ok {
		t.Fatalf("session.Page has type %T, want playwright.Page", session.Page)
	}
	title, err := page.Title()
	if err != nil {
		t.Fatalf("page.Title() error = %v", err)
	}
	if !strings.Contains(title, "Browser Probe") {
		t.Fatalf("page title = %q, want Browser Probe", title)
	}
}
