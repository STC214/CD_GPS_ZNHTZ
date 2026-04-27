package hentai2

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	summaryTitleRe   = regexp.MustCompile(`(?is)<h1[^>]*class=["'][^"']*\btitle\b[^"']*["'][^>]*>(.*?)</h1>`)
	summaryPagesRe   = regexp.MustCompile(`(?is)Pages\s*:\s*(?:</[^>]+>\s*)?(\d+)`)
	summaryReadRe    = regexp.MustCompile(`(?is)<a[^>]*href\s*=\s*["']([^"']*?/read/[^"']+?\.html[^"']*)["'][^>]*>.*?Read\s+Online.*?</a>`)
	summaryReadAnyRe = regexp.MustCompile(`(?is)href\s*=\s*["']([^"']*?/read/[^"']+?\.html[^"']*)["']`)
	tagRe            = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRe          = regexp.MustCompile(`\s+`)
)

// SummaryPage describes the important summary-page fields for Hentai2.
type SummaryPage struct {
	BaseURL    string   `json:"baseURL"`
	Title      string   `json:"title"`
	PageCount  int      `json:"pageCount"`
	ReaderURL  string   `json:"readerURL"`
	ReaderURLs []string `json:"readerURLs"`
}

// ParseSummaryPage parses a Hentai2 gallery summary page.
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
	for _, re := range []*regexp.Regexp{summaryReadRe, summaryReadAnyRe} {
		for _, match := range re.FindAllStringSubmatch(pageHTML, -1) {
			if len(match) < 2 {
				continue
			}
			candidate := resolveURL(baseURL, match[1])
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			resolved = append(resolved, candidate)
		}
		if len(resolved) > 0 {
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
