package hentaiaz

import (
	"fmt"
	"html"
	"log"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	readerTitleRe   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	readerBlockRe   = regexp.MustCompile(`(?is)<section[^>]*id=["']image-container["'][^>]*class=["'][^"']*\bread1\b[^"']*["'][^>]*>(.*?)</section>`)
	readerImgSrcRe  = regexp.MustCompile(`(?is)<img[^>]*src\s*=\s*["']([^"']+)["'][^>]*>`)
	readerDataSrcRe = regexp.MustCompile(`(?is)<img[^>]*(?:data-src|data-original|data-lazy-src)\s*=\s*["']([^"']+)["'][^>]*>`)
	numeric5Re      = regexp.MustCompile(`\d{5,}`)
)

// ReaderPage describes the important reader-page fields for Hentaiaz.
type ReaderPage struct {
	BaseURL           string   `json:"baseURL"`
	URL               string   `json:"url"`
	Title             string   `json:"title"`
	ImageURLs         []string `json:"imageURLs,omitempty"`
	FilteredImageURLs []string `json:"filteredImageURLs,omitempty"`
	SharedSignature   string   `json:"sharedSignature,omitempty"`
}

// ParseReaderPage parses a Hentaiaz reader page and extracts image candidates.
func ParseReaderPage(baseURL, readerURL, pageHTML string, expectedPageCount int) (ReaderPage, error) {
	baseURL = strings.TrimSpace(baseURL)
	readerURL = strings.TrimSpace(readerURL)
	pageHTML = strings.TrimSpace(pageHTML)
	if baseURL == "" {
		return ReaderPage{}, fmt.Errorf("reader base url is empty")
	}
	if readerURL == "" {
		return ReaderPage{}, fmt.Errorf("reader url is empty")
	}
	if pageHTML == "" {
		return ReaderPage{}, fmt.Errorf("reader html is empty")
	}

	title := cleanHTMLText(findFirstSubmatch(readerTitleRe, pageHTML))
	imageURLs := collectReaderImageURLs(baseURL, pageHTML)
	filtered, signature := filterReaderImageURLs(imageURLs, expectedPageCount)
	filtered = expandSequentialReaderImageURLs(filtered, expectedPageCount)
	log.Printf("hentaiaz reader parse: url=%s title=%q imageURLs=%d filtered=%d signature=%q expected=%d",
		readerURL,
		title,
		len(imageURLs),
		len(filtered),
		signature,
		expectedPageCount,
	)

	return ReaderPage{
		BaseURL:           baseURL,
		URL:               readerURL,
		Title:             title,
		ImageURLs:         imageURLs,
		FilteredImageURLs: filtered,
		SharedSignature:   signature,
	}, nil
}

func collectReaderImageURLs(baseURL, pageHTML string) []string {
	scope := pageHTML
	if block := findFirstSubmatch(readerBlockRe, pageHTML); strings.TrimSpace(block) != "" {
		scope = block
	}
	seen := map[string]struct{}{}
	var resolved []string
	for _, re := range []*regexp.Regexp{readerDataSrcRe, readerImgSrcRe} {
		for _, match := range re.FindAllStringSubmatch(scope, -1) {
			if len(match) < 2 {
				continue
			}
			candidate := resolveURL(baseURL, html.UnescapeString(match[1]))
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			resolved = append(resolved, candidate)
		}
	}
	return resolved
}

func filterReaderImageURLs(urls []string, expectedPageCount int) ([]string, string) {
	urls = normalizeUniqueStrings(urls)
	if len(urls) == 0 {
		return nil, ""
	}
	candidates := make([]string, 0, len(urls))
	for _, raw := range urls {
		lower := strings.ToLower(raw)
		if !strings.Contains(lower, "uploads") {
			continue
		}
		if !numeric5Re.MatchString(raw) {
			continue
		}
		candidates = append(candidates, raw)
	}
	if len(candidates) == 0 {
		return nil, ""
	}
	signature := inferSharedSignature(candidates, expectedPageCount)
	if signature == "" {
		return candidates, ""
	}
	filtered := make([]string, 0, len(candidates))
	for _, raw := range candidates {
		if strings.Contains(raw, signature) {
			filtered = append(filtered, raw)
		}
	}
	return filtered, signature
}

func expandSequentialReaderImageURLs(urls []string, expectedPageCount int) []string {
	urls = normalizeUniqueStrings(urls)
	if expectedPageCount <= len(urls) || expectedPageCount <= 1 || len(urls) != 1 {
		return urls
	}
	first := urls[0]
	parsed, err := url.Parse(first)
	if err != nil {
		return urls
	}
	dir, file := path.Split(parsed.Path)
	ext := path.Ext(file)
	base := strings.TrimSuffix(file, ext)
	if ext == "" || base == "" {
		return urls
	}
	n, err := strconv.Atoi(base)
	if err != nil || n != 1 {
		return urls
	}
	expanded := make([]string, 0, expectedPageCount)
	for i := 1; i <= expectedPageCount; i++ {
		next := *parsed
		next.Path = dir + strconv.Itoa(i) + ext
		expanded = append(expanded, next.String())
	}
	return expanded
}

func inferSharedSignature(urls []string, expectedPageCount int) string {
	type score struct {
		sig   string
		count int
	}
	counts := map[string]map[string]struct{}{}
	for _, raw := range urls {
		for _, sig := range uniqueNumericSignatures(raw) {
			hits := counts[sig]
			if hits == nil {
				hits = map[string]struct{}{}
				counts[sig] = hits
			}
			hits[raw] = struct{}{}
		}
	}
	if len(counts) == 0 {
		return ""
	}
	scores := make([]score, 0, len(counts))
	for sig, hits := range counts {
		scores = append(scores, score{sig: sig, count: len(hits)})
	}
	sort.Slice(scores, func(i, j int) bool {
		if expectedPageCount > 0 {
			di := absInt(scores[i].count - expectedPageCount)
			dj := absInt(scores[j].count - expectedPageCount)
			if di != dj {
				return di < dj
			}
		}
		if scores[i].count != scores[j].count {
			return scores[i].count > scores[j].count
		}
		if len(scores[i].sig) != len(scores[j].sig) {
			return len(scores[i].sig) > len(scores[j].sig)
		}
		return scores[i].sig < scores[j].sig
	})
	return scores[0].sig
}

func uniqueNumericSignatures(raw string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, sig := range numeric5Re.FindAllString(raw, -1) {
		if _, ok := seen[sig]; ok {
			continue
		}
		seen[sig] = struct{}{}
		out = append(out, sig)
	}
	return out
}

func normalizeUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(html.UnescapeString(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
