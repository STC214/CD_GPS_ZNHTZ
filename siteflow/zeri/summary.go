package zeri

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	lengthRe = regexp.MustCompile(`(?is)Length\s*:\s*(\d+)\s*pages`)
	rowRe    = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*\brow\b[^"']*["'][^>]*>(.*?)</div>`)
	hrefRe   = regexp.MustCompile(`(?is)href\s*=\s*["']([^"']+)["']`)
)

// SummaryPage describes the important summary-page fields for Zeri.
type SummaryPage struct {
	BaseURL        string   `json:"baseURL"`
	Title          string   `json:"title"`
	PageCount      int      `json:"pageCount"`
	LengthText     string   `json:"lengthText"`
	ReaderURL      string   `json:"readerURL"`
	ReaderURLs     []string `json:"readerURLs"`
	RowCount       int      `json:"rowCount"`
	ResolvedSource string   `json:"resolvedSource,omitempty"`
}

// ParseSummaryPage parses the summary page and resolves the first reader URL candidate.
func ParseSummaryPage(baseURL, pageHTML string) (SummaryPage, error) {
	baseURL = strings.TrimSpace(baseURL)
	pageHTML = strings.TrimSpace(pageHTML)
	if baseURL == "" {
		return SummaryPage{}, fmt.Errorf("summary base url is empty")
	}
	if pageHTML == "" {
		return SummaryPage{}, fmt.Errorf("summary html is empty")
	}

	title := strings.TrimSpace(findFirstSubmatch(titleRe, pageHTML))
	if title != "" {
		title = html.UnescapeString(title)
	}

	lengthText := strings.TrimSpace(findFirstSubmatch(lengthRe, pageHTML))
	pageCount := 0
	if lengthText != "" {
		pageCount = mustParseInt(lengthText)
	}

	readerURLs := collectReaderURLs(baseURL, pageHTML)
	readerURL := ""
	if len(readerURLs) > 0 {
		readerURL = readerURLs[0]
	}

	return SummaryPage{
		BaseURL:    baseURL,
		Title:      title,
		PageCount:  pageCount,
		LengthText: lengthText,
		ReaderURL:  readerURL,
		ReaderURLs: readerURLs,
		RowCount:   len(rowRe.FindAllStringSubmatch(pageHTML, -1)),
	}, nil
}

// ResolveReaderURLFromSummary parses the summary page and returns the first resolved reader URL.
func ResolveReaderURLFromSummary(baseURL, pageHTML string) (string, error) {
	page, err := ParseSummaryPage(baseURL, pageHTML)
	if err != nil {
		return "", err
	}
	if page.ReaderURL == "" {
		return "", fmt.Errorf("reader url not found in summary page")
	}
	return page.ReaderURL, nil
}

func collectReaderURLs(baseURL, pageHTML string) []string {
	rows := rowRe.FindAllStringSubmatch(pageHTML, -1)
	seen := make(map[string]struct{}, len(rows))
	resolved := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		for _, href := range hrefRe.FindAllStringSubmatch(row[1], -1) {
			if len(href) < 2 {
				continue
			}
			candidate := resolveURL(baseURL, href[1])
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			resolved = append(resolved, candidate)
			break
		}
	}
	return resolved
}

func resolveURL(baseURL, href string) string {
	href = strings.TrimSpace(html.UnescapeString(href))
	if href == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(href), "javascript:") || strings.HasPrefix(strings.ToLower(href), "mailto:") {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	next, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(next).String()
}

func findFirstSubmatch(re *regexp.Regexp, s string) string {
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func mustParseInt(text string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}
	n, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return n
}
