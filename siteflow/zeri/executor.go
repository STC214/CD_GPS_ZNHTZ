package zeri

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"comic_downloader_go_playwright_stealth/browser"
)

// ExecutionResult describes the resolved Zeri summary/reader flow.
type ExecutionResult struct {
	Summary                    SummaryPage  `json:"summary"`
	Reader                     ReaderPage   `json:"reader"`
	PaginationPages            []ReaderPage `json:"paginationPages,omitempty"`
	CollectedImages            []string     `json:"collectedImages,omitempty"`
	ActivationClicks           int          `json:"activationClicks"`
	PaginationActivationClicks int          `json:"paginationActivationClicks,omitempty"`
	FinalURL                   string       `json:"finalURL"`
	FinalTitle                 string       `json:"finalTitle"`
}

// IsZeriURL reports whether the URL belongs to the Zeri family we currently support.
func IsZeriURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return strings.Contains(host, "zeri-m.top") || strings.Contains(host, "zeri")
}

// Execute resolves the summary page, navigates to the reader page, and parses reader content.
func Execute(session browser.BrowserPageActions, summaryURL string) (ExecutionResult, error) {
	return ExecuteWithProgress(session, summaryURL, nil)
}

// ExecuteWithProgress resolves the summary page, navigates to the reader page, and reports overall progress as the flow advances.
func ExecuteWithProgress(session browser.BrowserPageActions, summaryURL string, progress DownloadProgressFunc) (ExecutionResult, error) {
	if session == nil {
		return ExecutionResult{}, fmt.Errorf("browser session is nil")
	}
	log.Printf("zeri execute start: summary=%s", summaryURL)
	report := func(current, total int, fraction float64, phase, message string) {
		if progress == nil {
			return
		}
		progress(DownloadProgress{
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
	log.Printf("zeri summary parsed: title=%q pages=%d readerURL=%s", summary.Title, summary.PageCount, summary.ReaderURL)
	if summary.ReaderURL == "" {
		return ExecutionResult{}, fmt.Errorf("reader url not found in summary page")
	}

	downloadWeight := computeDownloadWeight(summary.PageCount)
	parseWeight := 1.0 - downloadWeight
	if parseWeight < 0.2 {
		parseWeight = 0.2
	}

	report(0, 0, 0.10*parseWeight, "parse", "summary")
	if err := session.Goto(summary.ReaderURL); err != nil {
		return ExecutionResult{}, fmt.Errorf("goto reader url %q: %w", summary.ReaderURL, err)
	}
	log.Printf("zeri reader goto: %s", summary.ReaderURL)

	reader, err := hydrateReaderPage(session, summaryURL, summary.ReaderURL)
	if err != nil {
		return ExecutionResult{}, err
	}
	report(0, 0, 0.30*parseWeight, "parse", "reader")

	paginationPages, collectedImages, err := walkReaderPagination(session, summaryURL, reader, func(current, total int) {
		if progress == nil || total <= 0 {
			return
		}
		spanStart := 0.45 * parseWeight
		span := parseWeight - spanStart
		if span < 0 {
			span = 0
		}
		fraction := spanStart
		if total > 0 {
			fraction = spanStart + span*float64(current)/float64(total)
		}
		report(current, total, fraction, "parse", fmt.Sprintf("%d/%d", current, total))
	})
	if err != nil {
		return ExecutionResult{}, err
	}
	collectedImages = mergeUniqueStrings(append(reader.FilteredImageURLs, collectedImages...))
	log.Printf("zeri reader walk done: pages=%d images=%d", len(paginationPages), len(collectedImages))
	report(len(paginationPages), len(paginationPages), parseWeight, "parse", "done")

	return ExecutionResult{
		Summary:                    summary,
		Reader:                     reader,
		PaginationPages:            paginationPages,
		CollectedImages:            collectedImages,
		ActivationClicks:           0,
		PaginationActivationClicks: 0,
		FinalURL:                   reader.URL,
		FinalTitle:                 reader.Title,
	}, nil
}

func walkReaderPagination(session browser.BrowserPageActions, summaryURL string, first ReaderPage, progress func(current, total int)) ([]ReaderPage, []string, error) {
	paginationURLs := mergeUniqueStrings(first.PaginationURLs)
	paginationPages := make([]ReaderPage, 0, len(paginationURLs))
	collectedImages := make([]string, 0)
	total := len(paginationURLs)
	log.Printf("zeri pagination start: pages=%d first=%s", total, first.URL)

	for _, pageURL := range paginationURLs {
		if strings.TrimSpace(pageURL) == "" {
			continue
		}
		log.Printf("zeri pagination step: goto=%s", pageURL)
		if pageURL != first.URL {
			if err := session.Goto(pageURL); err != nil {
				return nil, nil, fmt.Errorf("goto pagination url %q: %w", pageURL, err)
			}
		}
		page, err := hydrateReaderPage(session, summaryURL, pageURL)
		if err != nil {
			return nil, nil, err
		}
		paginationPages = append(paginationPages, page)
		collectedImages = append(collectedImages, page.FilteredImageURLs...)
		log.Printf("zeri pagination parsed: url=%s title=%q imageURLs=%d filtered=%d", pageURL, page.Title, len(page.ImageURLs), len(page.FilteredImageURLs))
		if progress != nil {
			progress(len(paginationPages), total)
		}
	}

	return paginationPages, mergeUniqueStrings(collectedImages), nil
}

func hydrateReaderPage(session browser.BrowserPageActions, summaryURL, pageURL string) (ReaderPage, error) {
	initialHTML, err := session.Content()
	if err != nil {
		return ReaderPage{}, fmt.Errorf("read reader html %q: %w", pageURL, err)
	}
	page, err := ParseReaderPage(summaryURL, pageURL, initialHTML)
	if err != nil {
		return ReaderPage{}, err
	}
	log.Printf("zeri reader initial parse: url=%s title=%q imageURLs=%d filtered=%d has100=%t", pageURL, page.Title, len(page.ImageURLs), len(page.FilteredImageURLs), page.HasZoom100)

	expectedImages := len(page.FilteredImageURLs)
	if expectedImages <= 0 {
		expectedImages = len(page.ImageURLs)
	}
	log.Printf("zeri reader lazy wait: url=%s expectedImages=%d", pageURL, expectedImages)
	if err := session.LoadLazyContentForCount(expectedImages); err != nil {
		return ReaderPage{}, fmt.Errorf("load lazy content %q: %w", pageURL, err)
	}

	finalHTML, err := session.Content()
	if err != nil {
		return ReaderPage{}, fmt.Errorf("read activated reader html %q: %w", pageURL, err)
	}
	activatedPage, err := ParseReaderPage(summaryURL, pageURL, finalHTML)
	if err != nil {
		return ReaderPage{}, err
	}
	log.Printf("zeri reader activated parse: url=%s title=%q imageURLs=%d filtered=%d", pageURL, activatedPage.Title, len(activatedPage.ImageURLs), len(activatedPage.FilteredImageURLs))
	return activatedPage, nil
}

func computeDownloadWeight(targetCount int) float64 {
	if targetCount <= 0 {
		return 0.35
	}
	weight := 0.25 + float64(targetCount)/float64(targetCount+8)*0.50
	if weight < 0.25 {
		weight = 0.25
	}
	if weight > 0.75 {
		weight = 0.75
	}
	return weight
}

// DownloadWeightForCount returns the progress fraction reserved for the download phase.
func DownloadWeightForCount(targetCount int) float64 {
	return computeDownloadWeight(targetCount)
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

func mergeUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	merged := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}
	return merged
}
