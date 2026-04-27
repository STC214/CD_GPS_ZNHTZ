package nyahentai

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

	"comic_downloader_go_playwright_stealth/browser"
)

const (
	minImageWidth  = 300
	minImageHeight = 300
)

var (
	readerTitleRe   = regexp.MustCompile(`(?is)<div[^>]*id=["']post-data["'][^>]*>.*?<h1[^>]*>(.*?)</h1>`)
	postComicOpenRe = regexp.MustCompile(`(?is)<[^>]+id=["']post-comic["'][^>]*>`)
	readerImgTagRe  = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	readerImgAttrRe = regexp.MustCompile(`(?is)\b(?:data-src|data-original|data-lazy-src|data-srcset|srcset|src)\s*=\s*["']([^"']+)["']`)
	numeric6Re      = regexp.MustCompile(`\d{6,}`)
	htmlImageURLRe  = regexp.MustCompile(`(?is)https?://[^"'\s>]+?\.(?:jpe?g|png|gif|webp|bmp)(?:\?[^"'\s>]*)?`)
	tagRe           = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRe         = regexp.MustCompile(`\s+`)
)

// ReaderPage describes the important reader-page fields for Nyahentai.
type ReaderPage struct {
	BaseURL           string   `json:"baseURL"`
	URL               string   `json:"url"`
	Title             string   `json:"title"`
	ImageURLs         []string `json:"imageURLs,omitempty"`
	FilteredImageURLs []string `json:"filteredImageURLs,omitempty"`
	SharedSignature   string   `json:"sharedSignature,omitempty"`
}

// ParseReaderPage parses a Nyahentai reader page and extracts image candidates.
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

	signature := readerIDFromURL(readerURL)
	title := cleanHTMLText(findFirstSubmatch(readerTitleRe, pageHTML))
	imageURLs := collectReaderImageURLs(baseURL, pageHTML)
	filtered := filterReaderImageURLs(imageURLs, signature)
	log.Printf("nyahentai reader parse: url=%s title=%q imageURLs=%d filtered=%d signature=%q",
		readerURL,
		title,
		len(imageURLs),
		len(filtered),
		signature,
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
	scope := postComicScope(pageHTML)
	seen := map[string]struct{}{}
	var resolved []string
	for _, tag := range readerImgTagRe.FindAllString(scope, -1) {
		for _, match := range readerImgAttrRe.FindAllStringSubmatch(tag, -1) {
			if len(match) < 2 {
				continue
			}
			resolved = appendResolvedImageCandidates(resolved, seen, baseURL, match[1])
		}
	}
	return resolved
}

func postComicScope(pageHTML string) string {
	match := postComicOpenRe.FindStringIndex(pageHTML)
	if len(match) != 2 {
		return pageHTML
	}
	return pageHTML[match[0]:]
}

func appendResolvedImageCandidates(out []string, seen map[string]struct{}, baseURL, raw string) []string {
	for _, part := range strings.Split(raw, ",") {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		fields := strings.Fields(candidate)
		if len(fields) > 0 {
			candidate = fields[0]
		}
		resolved := resolveURL(baseURL, html.UnescapeString(candidate))
		if resolved == "" {
			continue
		}
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	return out
}

func filterReaderImageURLs(urls []string, signature string) []string {
	urls = normalizeUniqueStrings(urls)
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return nil
	}
	filtered := make([]string, 0, len(urls))
	for _, raw := range urls {
		if strings.Contains(raw, signature) {
			filtered = append(filtered, raw)
		}
	}
	return filtered
}

func collectReaderImageURLsFromRecords(baseURL, readerURL string, records []browser.PageImageRecord, fallbackHTML string) []string {
	signature := readerIDFromURL(readerURL)
	if signature == "" {
		return nil
	}
	var comicBig []string
	var comicUnsized []string
	var htmlURLs []string
	for _, record := range records {
		raw := strings.TrimSpace(record.Src)
		if raw == "" {
			continue
		}
		resolved := resolveURL(baseURL, raw)
		if resolved == "" || !isComicImageURL(resolved, signature) {
			continue
		}
		width, height := recordImageSize(record)
		if width >= minImageWidth && height >= minImageHeight {
			comicBig = append(comicBig, resolved)
			continue
		}
		if width == 0 || height == 0 {
			comicUnsized = append(comicUnsized, resolved)
			continue
		}
		if !isThumbnailHint(resolved) {
			comicUnsized = append(comicUnsized, resolved)
		}
	}
	if len(comicBig) > 0 || len(comicUnsized) > 0 {
		return sortComicURLs(append(comicBig, comicUnsized...), signature)
	}
	for _, raw := range collectReaderImageURLs(baseURL, fallbackHTML) {
		if isComicImageURL(raw, signature) && !isThumbnailHint(raw) {
			htmlURLs = append(htmlURLs, raw)
		}
	}
	for _, match := range htmlImageURLRe.FindAllString(fallbackHTML, -1) {
		resolved := resolveURL(baseURL, html.UnescapeString(match))
		if isComicImageURL(resolved, signature) && !isThumbnailHint(resolved) {
			htmlURLs = append(htmlURLs, resolved)
		}
	}
	return sortComicURLs(htmlURLs, signature)
}

func recordImageSize(record browser.PageImageRecord) (int, int) {
	width := firstPositiveInt(record.AttrWidth, record.NaturalWidth, record.OffsetWidth, record.ClientWidth, record.RectWidth)
	height := firstPositiveInt(record.AttrHeight, record.NaturalHeight, record.OffsetHeight, record.ClientHeight, record.RectHeight)
	return width, height
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func isThumbnailHint(raw string) bool {
	lower := strings.ToLower(raw)
	for _, token := range []string{"thumb", "thumbnail", "preview", "small", "cover", "avatar", "icon", "sprite"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func isComicImageURL(raw, signature string) bool {
	if strings.TrimSpace(signature) == "" {
		return true
	}
	return strings.Contains(raw, signature)
}

func sortComicURLs(urls []string, signature string) []string {
	urls = normalizeUniqueStrings(urls)
	sort.SliceStable(urls, func(i, j int) bool {
		ii := comicImageIndex(urls[i], signature)
		ij := comicImageIndex(urls[j], signature)
		if ii != ij {
			return ii < ij
		}
		return urls[i] < urls[j]
	})
	return urls
}

func comicImageIndex(raw, signature string) int {
	if signature == "" {
		return 1 << 30
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return 1 << 30
	}
	cleanPath := parsed.Path
	for _, pattern := range []string{
		`(?i)/galleries/` + regexp.QuoteMeta(signature) + `/(\d+)\.`,
		`(?i)/` + regexp.QuoteMeta(signature) + `/(\d+)\.`,
	} {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(cleanPath); len(match) > 1 {
			if n, err := strconv.Atoi(match[1]); err == nil {
				return n
			}
		}
	}
	base := strings.TrimSuffix(path.Base(cleanPath), path.Ext(cleanPath))
	if n, err := strconv.Atoi(strings.TrimLeft(base, "0")); err == nil {
		return n
	}
	return 1 << 30
}

func readerIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if parsed, err := url.Parse(raw); err == nil {
		for _, pattern := range []string{`(?i)/fanzine/re(\d+)`, `(?i)re(\d+)`} {
			re := regexp.MustCompile(pattern)
			if match := re.FindStringSubmatch(parsed.Path); len(match) > 1 {
				return match[1]
			}
		}
		for _, candidate := range numeric6Re.FindAllString(parsed.Path, -1) {
			if candidate != "" {
				return candidate
			}
		}
	}
	if match := regexp.MustCompile(`(?i)re(\d+)`).FindStringSubmatch(raw); len(match) > 1 {
		return match[1]
	}
	match := numeric6Re.FindString(raw)
	return match
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
