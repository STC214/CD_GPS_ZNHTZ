package nyahentai

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"comic_downloader_go_playwright_stealth/browser"
	"comic_downloader_go_playwright_stealth/siteflow/zeri"
)

// ExecutionResult describes the resolved Nyahentai reader flow.
type ExecutionResult struct {
	Reader          ReaderPage `json:"reader"`
	CollectedImages []string   `json:"collectedImages,omitempty"`
	FinalURL        string     `json:"finalURL"`
	FinalTitle      string     `json:"finalTitle"`
	PageCount       int        `json:"pageCount,omitempty"`
}

// IsNyahentaiURL reports whether the URL belongs to Nyahentai.
func IsNyahentaiURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return strings.Contains(host, "nyahentai")
}

// ExecuteWithProgress reads the current reader page directly and reports progress.
func ExecuteWithProgress(session browser.BrowserPageActions, readerURL string, progress zeri.DownloadProgressFunc) (ExecutionResult, error) {
	if session == nil {
		return ExecutionResult{}, fmt.Errorf("browser session is nil")
	}
	log.Printf("nyahentai execute start: reader=%s", readerURL)
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

	report(0, 0, 0.02, "start", "reader")
	reader, err := hydrateReaderPage(session, readerURL)
	if err != nil {
		return ExecutionResult{}, err
	}
	pageCount := len(reader.FilteredImageURLs)
	if pageCount == 0 {
		return ExecutionResult{}, fmt.Errorf("nyahentai target images not found: reader=%s id=%s candidates=%d", readerURL, reader.SharedSignature, len(reader.ImageURLs))
	}
	report(pageCount, pageCount, 1, "parse", "done")

	return ExecutionResult{
		Reader:          reader,
		CollectedImages: reader.FilteredImageURLs,
		FinalURL:        reader.URL,
		FinalTitle:      reader.Title,
		PageCount:       pageCount,
	}, nil
}

func hydrateReaderPage(session browser.BrowserPageActions, pageURL string) (ReaderPage, error) {
	initialHTML, err := session.Content()
	if err != nil {
		return ReaderPage{}, fmt.Errorf("read reader html %q: %w", pageURL, err)
	}
	initialPage, err := ParseReaderPage(pageURL, pageURL, initialHTML)
	if err != nil {
		return ReaderPage{}, err
	}
	log.Printf("nyahentai reader lazy wait: url=%s initialImages=%d filtered=%d", pageURL, len(initialPage.ImageURLs), len(initialPage.FilteredImageURLs))
	if err := loadPostComicLazyContent(session); err != nil {
		return ReaderPage{}, fmt.Errorf("load post-comic lazy content %q: %w", pageURL, err)
	}

	finalHTML, err := session.Content()
	if err != nil {
		return ReaderPage{}, fmt.Errorf("read activated reader html %q: %w", pageURL, err)
	}
	activatedPage, err := ParseReaderPage(pageURL, pageURL, finalHTML)
	if err != nil {
		return ReaderPage{}, err
	}
	if records, recordErr := imageRecords(session); recordErr == nil && len(records) > 0 {
		if imageURLs := collectReaderImageURLsFromRecords(pageURL, pageURL, records, finalHTML); len(imageURLs) > 0 {
			activatedPage.ImageURLs = imageURLs
			activatedPage.FilteredImageURLs = imageURLs
			activatedPage.SharedSignature = readerIDFromURL(pageURL)
		}
	} else if recordErr != nil {
		log.Printf("nyahentai image records unavailable: %v", recordErr)
	}
	log.Printf("nyahentai reader activated parse: url=%s title=%q imageURLs=%d filtered=%d", pageURL, activatedPage.Title, len(activatedPage.ImageURLs), len(activatedPage.FilteredImageURLs))
	return activatedPage, nil
}

func imageRecords(session browser.BrowserPageActions) ([]browser.PageImageRecord, error) {
	type imageRecorder interface {
		ImageRecords() ([]browser.PageImageRecord, error)
	}
	if recorder, ok := session.(imageRecorder); ok {
		return recorder.ImageRecords()
	}
	return nil, fmt.Errorf("image records are not supported")
}

func loadPostComicLazyContent(session browser.BrowserPageActions) error {
	type selectorLazyLoader interface {
		LoadLazyContentInSelector(selector string) error
	}
	if loader, ok := session.(selectorLazyLoader); ok {
		return loader.LoadLazyContentInSelector("#post-comic")
	}
	return session.LoadLazyContent()
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
