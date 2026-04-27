package hentaiaz

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	summaryTitleRe = regexp.MustCompile(`(?is)<span[^>]*class=["'][^"']*\btext-pink\b[^"']*["'][^>]*>(.*?)</span>`)
	summaryPagesRe = regexp.MustCompile(`(?is)<i[^>]*class=["'][^"']*\bfa\b[^"']*\bfa-file\b[^"']*["'][^>]*>\s*</i>\s*(\d{1,4})\s*pages\b`)
	summaryReadRe  = regexp.MustCompile(`(?is)<a[^>]*title\s*=\s*["']\s*List\s+Read\b[^"']*["'][^>]*href\s*=\s*["']([^"']*?/read/[^"']+?\.html[^"']*)["'][^>]*>|<a[^>]*href\s*=\s*["']([^"']*?/read/[^"']+?\.html[^"']*)["'][^>]*title\s*=\s*["']\s*List\s+Read\b[^"']*["'][^>]*>`)
	tagRe          = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRe        = regexp.MustCompile(`\s+`)
)

// SummaryPage describes the important summary-page fields for Hentaiaz.
type SummaryPage struct {
	BaseURL    string   `json:"baseURL"`
	Title      string   `json:"title"`
	PageCount  int      `json:"pageCount"`
	ReaderURL  string   `json:"readerURL"`
	ReaderURLs []string `json:"readerURLs"`
}

// ParseSummaryPage parses a Hentaiaz gallery summary page.
func ParseSummaryPage(baseURL, pageHTML string) (SummaryPage, error) {
	baseURL = strings.TrimSpace(baseURL)
	pageHTML = strings.TrimSpace(pageHTML)
	if baseURL == "" {
		return SummaryPage{}, fmt.Errorf("summary base url is empty")
	}
	if pageHTML == "" {
		return SummaryPage{}, fmt.Errorf("summary html is empty")
	}

	title := cleanHTMLText(findFirstSubmatch(summaryTitleRe, pageHTML))
	pageCount := mustParseInt(findFirstSubmatch(summaryPagesRe, pageHTML))
	readerURLs := collectSummaryReaderURLs(baseURL, pageHTML)
	readerURL := ""
	if len(readerURLs) > 0 {
		readerURL = readerURLs[0]
	}

	return SummaryPage{
		BaseURL:    baseURL,
		Title:      title,
		PageCount:  pageCount,
		ReaderURL:  readerURL,
		ReaderURLs: readerURLs,
	}, nil
}

func collectSummaryReaderURLs(baseURL, pageHTML string) []string {
	seen := map[string]struct{}{}
	var resolved []string
	for _, match := range summaryReadRe.FindAllStringSubmatch(pageHTML, -1) {
		href := ""
		for i := 1; i < len(match); i++ {
			if strings.TrimSpace(match[i]) != "" {
				href = match[i]
				break
			}
		}
		candidate := resolveURL(baseURL, href)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		resolved = append(resolved, candidate)
	}
	return resolved
}

func resolveURL(baseURL, href string) string {
	href = strings.TrimSpace(html.UnescapeString(href))
	if href == "" {
		return ""
	}
	lower := strings.ToLower(href)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "mailto:") {
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

func cleanHTMLText(text string) string {
	text = strings.TrimSpace(html.UnescapeString(text))
	if text == "" {
		return ""
	}
	text = tagRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(html.UnescapeString(text))
	return spaceRe.ReplaceAllString(text, " ")
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
