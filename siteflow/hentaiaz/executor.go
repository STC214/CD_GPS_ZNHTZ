package hentaiaz

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"comic_downloader_go_playwright_stealth/browser"
	"comic_downloader_go_playwright_stealth/siteflow/zeri"
)

// ExecutionResult describes the resolved Hentaiaz summary/reader flow.
type ExecutionResult struct {
	Summary         SummaryPage `json:"summary"`
	Reader          ReaderPage  `json:"reader"`
	CollectedImages []string    `json:"collectedImages,omitempty"`
	FinalURL        string      `json:"finalURL"`
	FinalTitle      string      `json:"finalTitle"`
}

// IsHentaiazURL reports whether the URL belongs to Hentaiaz.
func IsHentaiazURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return strings.Contains(host, "hentaiaz")
}

// ExecuteWithProgress resolves the summary page, navigates to the reader page, and reports progress.
func ExecuteWithProgress(session browser.BrowserPageActions, summaryURL string, progress zeri.DownloadProgressFunc) (ExecutionResult, error) {
	if session == nil {
		return ExecutionResult{}, fmt.Errorf("browser session is nil")
	}
	log.Printf("hentaiaz execute start: summary=%s", summaryURL)
	report := func(current, total int, fraction float64, phase, message string) {
		if progress == nil {
			return
		}
		progress(zeri.DownloadProgress{
			Current:  current,
			Total:    total,
			Fraction: clamp01(fraction),
			Phase:    phase,
			Message:  message,
		})
	}

	report(0, 0, 0.02, "start", "summary")
	summaryHTML, err := session.Content()
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("read summary html: %w", err)
	}
	summary, err := ParseSummaryPage(summaryURL, summaryHTML)
	if err != nil {
		return ExecutionResult{}, err
	}
	log.Printf("hentaiaz summary parsed: title=%q pages=%d readerURL=%s", summary.Title, summary.PageCount, summary.ReaderURL)
	if summary.ReaderURL == "" {
		return ExecutionResult{}, fmt.Errorf("reader url not found in summary page")
	}

	downloadWeight := zeri.DownloadWeightForCount(summary.PageCount)
	parseWeight := 1.0 - downloadWeight
	if parseWeight < 0.2 {
		parseWeight = 0.2
	}

	report(0, 0, 0.10*parseWeight, "parse", "summary")
	if err := session.Goto(summary.ReaderURL); err != nil {
		return ExecutionResult{}, fmt.Errorf("goto reader url %q: %w", summary.ReaderURL, err)
	}
	log.Printf("hentaiaz reader goto: %s", summary.ReaderURL)

	reader, err := hydrateReaderPage(session, summary.ReaderURL, summary.PageCount)
	if err != nil {
		return ExecutionResult{}, err
	}
	report(len(reader.FilteredImageURLs), summary.PageCount, parseWeight, "parse", "done")

	return ExecutionResult{
		Summary:         summary,
		Reader:          reader,
		CollectedImages: reader.FilteredImageURLs,
		FinalURL:        reader.URL,
		FinalTitle:      reader.Title,
	}, nil
}

func hydrateReaderPage(session browser.BrowserPageActions, pageURL string, expectedPageCount int) (ReaderPage, error) {
	initialHTML, err := session.Content()
	if err != nil {
		return ReaderPage{}, fmt.Errorf("read reader html %q: %w", pageURL, err)
	}
	page, err := ParseReaderPage(pageURL, pageURL, initialHTML, expectedPageCount)
	if err != nil {
		return ReaderPage{}, err
	}
	expectedImages := expectedPageCount
	if expectedImages <= 0 {
		expectedImages = len(page.FilteredImageURLs)
	}
	if expectedImages <= 0 {
		expectedImages = len(page.ImageURLs)
	}
	log.Printf("hentaiaz reader lazy wait: url=%s expectedImages=%d", pageURL, expectedImages)
	if err := session.LoadLazyContentForCount(expectedImages); err != nil {
		return ReaderPage{}, fmt.Errorf("load lazy content %q: %w", pageURL, err)
	}

	finalHTML, err := session.Content()
	if err != nil {
		return ReaderPage{}, fmt.Errorf("read activated reader html %q: %w", pageURL, err)
	}
	activatedPage, err := ParseReaderPage(pageURL, pageURL, finalHTML, expectedPageCount)
	if err != nil {
		return ReaderPage{}, err
	}
	log.Printf("hentaiaz reader activated parse: url=%s title=%q imageURLs=%d filtered=%d", pageURL, activatedPage.Title, len(activatedPage.ImageURLs), len(activatedPage.FilteredImageURLs))
	return activatedPage, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
