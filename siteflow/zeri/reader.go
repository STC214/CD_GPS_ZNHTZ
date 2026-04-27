package zeri

import (
	"fmt"
	"html"
	"log"
	"regexp"
	"sort"
	"strings"
)

var (
	readerTitleRe     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	readerPageCountRe = regexp.MustCompile(`(?is)Length\s*:\s*(\d+)\s*pages`)
	readerZoom100Re   = regexp.MustCompile(`(?is)(?:^|[^0-9])100%(?:[^0-9]|$)`)
	readerBlockRe     = regexp.MustCompile(`(?is)<[^>]*id=["']([^"']+)["'][^>]*>(.*?)</[^>]+>`)
	readerHrefRe      = regexp.MustCompile(`(?is)href\s*=\s*["']([^"']+)["']`)
	readerImgSrcRe    = regexp.MustCompile(`(?is)<img[^>]*src\s*=\s*["']([^"']+)["'][^>]*>`)
	readerDataSrcRe   = regexp.MustCompile(`(?is)<img[^>]*(?:data-src|data-original)\s*=\s*["']([^"']+)["'][^>]*>`)
)

// ReaderPage describes the important reader-page fields for Zeri.
type ReaderPage struct {
	BaseURL           string   `json:"baseURL"`
	URL               string   `json:"url"`
	Title             string   `json:"title"`
	PageCount         int      `json:"pageCount"`
	LengthText        string   `json:"lengthText"`
	HasZoom100        bool     `json:"hasZoom100"`
	Zoom100Clicks     int      `json:"zoom100Clicks,omitempty"`
	ReaderURLs        []string `json:"readerURLs,omitempty"`
	PaginationURLs    []string `json:"paginationURLs,omitempty"`
	ImageURLs         []string `json:"imageURLs,omitempty"`
	FilteredImageURLs []string `json:"filteredImageURLs,omitempty"`
	SharedSignatures  []string `json:"sharedSignatures,omitempty"`
}

// ParseReaderPage parses the reader page and extracts the key navigation and image candidates.
func ParseReaderPage(baseURL, readerURL, pageHTML string) (ReaderPage, error) {
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

	title := strings.TrimSpace(findFirstSubmatch(readerTitleRe, pageHTML))
	if title != "" {
		title = html.UnescapeString(title)
	}
	lengthText := strings.TrimSpace(findFirstSubmatch(readerPageCountRe, pageHTML))
	pageCount := mustParseInt(lengthText)

	readerURLs := collectReaderURLs(baseURL, pageHTML)
	paginationURLs := collectReaderPaginationURLs(baseURL, pageHTML)
	imageURLs := collectReaderImageURLs(baseURL, pageHTML)
	signatures := InferReaderSignaturePair(imageURLs, 6)
	filteredImageURLs := FilterReaderImageURLsBySignatures(imageURLs, signatures)
	activation := ParseReaderActivationHint(pageHTML)
	log.Printf("zeri reader parse: url=%s title=%q pages=%d readerURLs=%d paginationURLs=%d imageURLs=%d filtered=%d signatures=%v has100=%t clickCount=%d",
		readerURL,
		title,
		pageCount,
		len(readerURLs),
		len(paginationURLs),
		len(imageURLs),
		len(filteredImageURLs),
		signatures,
		activation.HasZoom100,
		activation.ClickCount,
	)

	return ReaderPage{
		BaseURL:           baseURL,
		URL:               readerURL,
		Title:             title,
		PageCount:         pageCount,
		LengthText:        lengthText,
		HasZoom100:        activation.HasZoom100,
		Zoom100Clicks:     activation.ClickCount,
		ReaderURLs:        readerURLs,
		PaginationURLs:    paginationURLs,
		ImageURLs:         imageURLs,
		FilteredImageURLs: filteredImageURLs,
		SharedSignatures:  signatures,
	}, nil
}

// ParseReaderURLs returns all resolved reader candidates from the summary page HTML.
func ParseReaderURLs(baseURL, pageHTML string) ([]string, error) {
	page, err := ParseSummaryPage(baseURL, pageHTML)
	if err != nil {
		return nil, err
	}
	if len(page.ReaderURLs) == 0 {
		return nil, fmt.Errorf("reader url not found in summary page")
	}
	return page.ReaderURLs, nil
}

// ParseReaderPaginationURLs extracts merged and deduplicated pagination URLs from a reader page.
func ParseReaderPaginationURLs(baseURL, pageHTML string) ([]string, error) {
	baseURL = strings.TrimSpace(baseURL)
	pageHTML = strings.TrimSpace(pageHTML)
	if baseURL == "" {
		return nil, fmt.Errorf("reader base url is empty")
	}
	if pageHTML == "" {
		return nil, fmt.Errorf("reader html is empty")
	}
	return collectReaderPaginationURLs(baseURL, pageHTML), nil
}

// ParseReaderImageURLs extracts candidate image URLs from a reader page.
func ParseReaderImageURLs(baseURL, pageHTML string) ([]string, error) {
	baseURL = strings.TrimSpace(baseURL)
	pageHTML = strings.TrimSpace(pageHTML)
	if baseURL == "" {
		return nil, fmt.Errorf("reader base url is empty")
	}
	if pageHTML == "" {
		return nil, fmt.Errorf("reader html is empty")
	}
	return collectReaderImageURLs(baseURL, pageHTML), nil
}

// ParseReaderActivationHint returns a minimal hint for the browser layer to
// activate reader-mode zooming.
type ReaderActivationHint struct {
	HasZoom100 bool   `json:"hasZoom100"`
	ClickText  string `json:"clickText"`
	ClickCount int    `json:"clickCount"`
}

// ParseReaderActivationHint scans the page HTML for 100% activation points.
func ParseReaderActivationHint(pageHTML string) ReaderActivationHint {
	pageHTML = strings.TrimSpace(pageHTML)
	if pageHTML == "" {
		return ReaderActivationHint{ClickText: "100%"}
	}
	occurrences := len(readerZoom100Re.FindAllStringIndex(pageHTML, -1))
	return ReaderActivationHint{
		HasZoom100: occurrences > 0,
		ClickText:  "100%",
		ClickCount: occurrences,
	}
}

func collectReaderPaginationURLs(baseURL, pageHTML string) []string {
	seen := make(map[string]struct{})
	resolved := make([]string, 0)
	for _, href := range readerHrefRe.FindAllStringSubmatch(pageHTML, -1) {
		if len(href) < 2 {
			continue
		}
		raw := strings.TrimSpace(href[1])
		lower := strings.ToLower(raw)
		if !strings.Contains(lower, "page=") && !strings.Contains(lower, "page_num") {
			continue
		}
		candidate := resolveURL(baseURL, raw)
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

func collectReaderImageURLs(baseURL, pageHTML string) []string {
	seen := make(map[string]struct{})
	resolved := make([]string, 0)
	for _, re := range []*regexp.Regexp{readerDataSrcRe, readerImgSrcRe} {
		for _, src := range re.FindAllStringSubmatch(pageHTML, -1) {
			if len(src) < 2 {
				continue
			}
			candidate := resolveURL(baseURL, src[1])
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

// FilterReaderImageURLsBySignatures keeps only URLs that contain all shared signatures.
func FilterReaderImageURLsBySignatures(urls []string, signatures []string) []string {
	urls = normalizeUniqueStrings(urls)
	signatures = normalizeUniqueStrings(signatures)
	if len(urls) == 0 {
		return nil
	}
	if len(signatures) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(urls))
	for _, candidate := range urls {
		if containsAllSignatures(candidate, signatures) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

// FilterReaderImageURLs returns both shared signatures and the filtered final image list.
func FilterReaderImageURLs(urls []string, minDigits int) (filtered []string, sharedSignatures []string) {
	urls = normalizeUniqueStrings(urls)
	if len(urls) == 0 {
		return nil, nil
	}
	shared := InferReaderSignaturePair(urls, minDigits)
	return FilterReaderImageURLsBySignatures(urls, shared), shared
}

// SharedNumericSignatures returns the numeric signatures shared by all URLs.
// Only signatures that appear in every URL and have at least minDigits digits are returned.
func SharedNumericSignatures(urls []string, minDigits int) []string {
	if len(urls) == 0 {
		return nil
	}
	type counts struct {
		total int
	}
	hits := map[string]counts{}
	for _, raw := range urls {
		found := uniqueNumericSignatures(raw, minDigits)
		for sig := range found {
			c := hits[sig]
			c.total++
			hits[sig] = c
		}
	}
	shares := make([]string, 0, len(hits))
	for sig, c := range hits {
		if c.total == len(urls) {
			shares = append(shares, sig)
		}
	}
	sort.Slice(shares, func(i, j int) bool {
		if len(shares[i]) == len(shares[j]) {
			return shares[i] < shares[j]
		}
		return len(shares[i]) < len(shares[j])
	})
	return shares
}

// InferReaderSignaturePair picks the most likely pair of numeric signatures shared by the
// reader images, favoring the pair that appears together in the largest number of URLs.
func InferReaderSignaturePair(urls []string, minDigits int) []string {
	urls = normalizeUniqueStrings(urls)
	if len(urls) == 0 {
		return nil
	}
	type pairScore struct {
		left  string
		right string
		count int
	}
	scores := map[string]*pairScore{}
	for _, raw := range urls {
		sigs := sortedUniqueSlice(uniqueNumericSignatures(raw, minDigits))
		if len(sigs) < 2 {
			continue
		}
		for i := 0; i < len(sigs)-1; i++ {
			for j := i + 1; j < len(sigs); j++ {
				left, right := sigs[i], sigs[j]
				key := left + "\x00" + right
				score := scores[key]
				if score == nil {
					score = &pairScore{left: left, right: right}
					scores[key] = score
				}
				score.count++
			}
		}
	}
	if len(scores) == 0 {
		return SharedNumericSignatures(urls, minDigits)
	}
	best := make([]pairScore, 0, len(scores))
	for _, score := range scores {
		best = append(best, *score)
	}
	sort.Slice(best, func(i, j int) bool {
		if best[i].count != best[j].count {
			return best[i].count > best[j].count
		}
		if len(best[i].left) != len(best[j].left) {
			return len(best[i].left) > len(best[j].left)
		}
		if len(best[i].right) != len(best[j].right) {
			return len(best[i].right) > len(best[j].right)
		}
		if best[i].left != best[j].left {
			return best[i].left < best[j].left
		}
		return best[i].right < best[j].right
	})
	return []string{best[0].left, best[0].right}
}

func containsAllSignatures(raw string, signatures []string) bool {
	if len(signatures) == 0 {
		return false
	}
	for _, sig := range signatures {
		if !strings.Contains(raw, sig) {
			return false
		}
	}
	return true
}

func normalizeUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(html.UnescapeString(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func uniqueNumericSignatures(raw string, minDigits int) map[string]struct{} {
	result := make(map[string]struct{})
	if minDigits < 1 {
		minDigits = 1
	}
	re := regexp.MustCompile(`\d{` + fmt.Sprintf("%d", minDigits) + `,}`)
	for _, sig := range re.FindAllString(raw, -1) {
		result[sig] = struct{}{}
	}
	return result
}

func sortedUniqueSlice(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	sorted := make([]string, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Slice(sorted, func(i, j int) bool {
		if len(sorted[i]) == len(sorted[j]) {
			return sorted[i] < sorted[j]
		}
		return len(sorted[i]) < len(sorted[j])
	})
	return sorted
}
