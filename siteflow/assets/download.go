package assets

import (
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// CollectionSummary is the site-neutral metadata needed to name a download batch.
type CollectionSummary struct {
	Site      string `json:"site,omitempty"`
	BaseURL   string `json:"baseURL,omitempty"`
	Title     string `json:"title"`
	PageCount int    `json:"pageCount,omitempty"`
	ReaderURL string `json:"readerURL,omitempty"`
}

// DownloadProgress reports the status of reader image downloads.
type DownloadProgress struct {
	Current  int     `json:"current"`
	Total    int     `json:"total"`
	Phase    string  `json:"phase,omitempty"`
	Message  string  `json:"message,omitempty"`
	Fraction float64 `json:"fraction"`
}

// DownloadProgressFunc receives download progress updates.
type DownloadProgressFunc func(DownloadProgress)

// DownloadResult summarizes a batch of downloaded reader images.
type DownloadResult struct {
	OutputDir string   `json:"outputDir"`
	Files     []string `json:"files,omitempty"`
	Bytes     int64    `json:"bytes"`
}

// DownloadImages downloads the collected image URLs into a title-scoped output directory.
func DownloadImages(summary CollectionSummary, imageURLs []string, outputRoot string, progress DownloadProgressFunc) (DownloadResult, error) {
	outputRoot = strings.TrimSpace(outputRoot)
	if outputRoot == "" {
		return DownloadResult{}, fmt.Errorf("output root is empty")
	}
	imageURLs = NormalizeUniqueStrings(imageURLs)
	if len(imageURLs) == 0 {
		return DownloadResult{}, fmt.Errorf("image urls are empty")
	}

	chapterDir := outputRoot
	if title := SanitizePathPart(summary.Title); title != "" {
		chapterDir = filepath.Join(chapterDir, title)
	}
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		return DownloadResult{}, fmt.Errorf("create output dir %q: %w", chapterDir, err)
	}
	log.Printf("asset download resolved dir: site=%s title=%q outputRoot=%s chapterDir=%s images=%d", summary.Site, summary.Title, outputRoot, chapterDir, len(imageURLs))

	files := make([]string, 0, len(imageURLs))
	var totalBytes int64
	usedNames := make(map[string]int, len(imageURLs))
	report := func(current int, phase, message string) {
		if progress == nil {
			return
		}
		total := len(imageURLs)
		fraction := 0.0
		if total > 0 {
			fraction = float64(current) / float64(total)
		}
		progress(DownloadProgress{
			Current:  current,
			Total:    total,
			Phase:    phase,
			Message:  message,
			Fraction: fraction,
		})
	}

	report(0, "downloading", "prepare")
	for i, raw := range imageURLs {
		log.Printf("asset download image start: site=%s %d/%d url=%s", summary.Site, i+1, len(imageURLs), raw)
		report(i, "downloading", fmt.Sprintf("%d/%d", i, len(imageURLs)))
		saved, written, err := downloadOneImage(raw, chapterDir, i+1, usedNames)
		if err != nil {
			return DownloadResult{}, err
		}
		files = append(files, saved)
		totalBytes += written
		log.Printf("asset download image done: site=%s %d/%d file=%s bytes=%d", summary.Site, i+1, len(imageURLs), saved, written)
		report(i+1, "downloading", fmt.Sprintf("%d/%d", i+1, len(imageURLs)))
	}
	report(len(imageURLs), "done", "download complete")

	log.Printf("asset download complete: site=%s files=%d bytes=%d dir=%s", summary.Site, len(files), totalBytes, chapterDir)
	return DownloadResult{
		OutputDir: chapterDir,
		Files:     files,
		Bytes:     totalBytes,
	}, nil
}

func downloadOneImage(rawURL, outputDir string, index int, usedNames map[string]int) (string, int64, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", 0, fmt.Errorf("image url is empty")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", 0, fmt.Errorf("parse image url %q: %w", rawURL, err)
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create image request %q: %w", rawURL, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("download image %q: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("download image %q: unexpected status %s", rawURL, resp.Status)
	}

	baseName := strings.TrimSuffix(filepath.Base(parsed.Path), filepath.Ext(parsed.Path))
	ext := sanitizeExt(filepath.Ext(parsed.Path))
	if ext == "" {
		ext = extFromContentType(resp.Header.Get("Content-Type"))
	}
	if ext == "" {
		ext = ".jpg"
	}

	filename := uniqueDownloadFilename(baseName, ext, index, usedNames)
	targetPath := filepath.Join(outputDir, filename)

	file, err := os.Create(targetPath)
	if err != nil {
		return "", 0, fmt.Errorf("create image file %q: %w", targetPath, err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("write image file %q: %w", targetPath, err)
	}
	return targetPath, written, nil
}

func uniqueDownloadFilename(baseName, ext string, index int, usedNames map[string]int) string {
	baseName = SanitizePathPart(baseName)
	if baseName == "" {
		baseName = fmt.Sprintf("%03d", index)
	}
	ext = sanitizeExt(ext)
	if ext == "" {
		ext = ".jpg"
	}
	key := strings.ToLower(baseName + ext)
	usedNames[key]++
	if usedNames[key] == 1 {
		return baseName + ext
	}
	return fmt.Sprintf("%s-%d%s", baseName, usedNames[key], ext)
}

func extFromContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	if ext, _ := mime.ExtensionsByType(contentType); len(ext) > 0 {
		for _, candidate := range ext {
			if candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func sanitizeExt(ext string) string {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		return ""
	}
	if len(ext) > 8 {
		return ""
	}
	if strings.ContainsAny(ext, `\/:*?"<>|`) {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		return ""
	}
	return strings.ToLower(ext)
}

// SanitizePathPart converts a site title or URL path part into a safe directory/file fragment.
func SanitizePathPart(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	count := 0
	lastUnderscore := false
	for _, r := range text {
		if count >= 64 {
			break
		}
		out := '_'
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r):
			out = r
		case r == ' ' || r == '-' || r == '_' || r == '.':
			out = '_'
		default:
			out = '_'
		}
		if out == '_' {
			if lastUnderscore {
				continue
			}
			lastUnderscore = true
		} else {
			lastUnderscore = false
		}
		b.WriteRune(out)
		count++
	}
	return strings.Trim(b.String(), "_")
}

// SelectThumbnailSource picks the best downloaded file for the task thumbnail.
func SelectThumbnailSource(files []string) string {
	files = NormalizeUniqueStrings(files)
	if len(files) == 0 {
		return ""
	}

	exact := make([]string, 0, 2)
	type numericCandidate struct {
		path  string
		value int
	}
	numeric := make([]numericCandidate, 0, len(files))

	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		switch base {
		case "1", "01":
			exact = append(exact, file)
			continue
		}

		digits := strings.TrimLeft(base, "0")
		if digits == "" {
			digits = "0"
		}
		value, err := strconv.Atoi(digits)
		if err != nil || value <= 0 {
			continue
		}
		numeric = append(numeric, numericCandidate{path: file, value: value})
	}

	if len(exact) > 0 {
		sort.SliceStable(exact, func(i, j int) bool {
			li := len(strings.TrimSuffix(filepath.Base(exact[i]), filepath.Ext(exact[i])))
			lj := len(strings.TrimSuffix(filepath.Base(exact[j]), filepath.Ext(exact[j])))
			if li != lj {
				return li < lj
			}
			return exact[i] < exact[j]
		})
		return exact[0]
	}

	if len(numeric) == 0 {
		return files[0]
	}
	sort.SliceStable(numeric, func(i, j int) bool {
		if numeric[i].value != numeric[j].value {
			return numeric[i].value < numeric[j].value
		}
		return numeric[i].path < numeric[j].path
	})
	return numeric[0].path
}

// NormalizeUniqueStrings trims, unescapes, and deduplicates strings in insertion order.
func NormalizeUniqueStrings(values []string) []string {
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
