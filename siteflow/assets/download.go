package assets

import (
	"context"
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
	"sync"
	"time"
	"unicode"
)

const maxDownloadWorkers = 7
const defaultImageDownloadAttempts = 4
const hitomiImageDownloadAttempts = 20

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
	return DownloadImagesContext(context.Background(), summary, imageURLs, outputRoot, progress)
}

// DownloadImagesContext downloads the collected image URLs and stops early when ctx is canceled.
func DownloadImagesContext(ctx context.Context, summary CollectionSummary, imageURLs []string, outputRoot string, progress DownloadProgressFunc) (DownloadResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	outputRoot = strings.TrimSpace(outputRoot)
	if outputRoot == "" {
		return DownloadResult{}, fmt.Errorf("output root is empty")
	}
	if err := ctx.Err(); err != nil {
		return DownloadResult{}, err
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

	files := make([]string, len(imageURLs))
	var totalBytes int64
	usedNames := make(map[string]int, len(imageURLs))
	var usedNamesMu sync.Mutex
	var progressMu sync.Mutex
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
	workerCount := downloadWorkerCount(len(imageURLs))
	log.Printf("asset download begin: site=%s images=%d workers=%d", summary.Site, len(imageURLs), workerCount)

	type job struct {
		index int
		url   string
	}
	jobs := make(chan job)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	completed := 0

	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
			cancel()
		}
		errMu.Unlock()
	}

	for workerID := 1; workerID <= workerCount; workerID++ {
		workerID := workerID
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-jobs:
					if !ok {
						return
					}
					log.Printf("asset download image start: site=%s worker=%d %d/%d url=%s", summary.Site, workerID, item.index+1, len(imageURLs), item.url)
					saved, written, err := downloadOneImage(ctx, item.url, chapterDir, item.index+1, &usedNamesMu, usedNames)
					if err != nil {
						setErr(err)
						return
					}
					progressMu.Lock()
					files[item.index] = saved
					totalBytes += written
					completed++
					current := completed
					progressMu.Unlock()
					log.Printf("asset download image done: site=%s worker=%d %d/%d file=%s bytes=%d", summary.Site, workerID, item.index+1, len(imageURLs), saved, written)
					report(current, "downloading", fmt.Sprintf("%d/%d", current, len(imageURLs)))
				}
			}
		}()
	}

	for i, raw := range imageURLs {
		select {
		case <-ctx.Done():
			break
		case jobs <- job{index: i, url: raw}:
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return DownloadResult{}, firstErr
	}
	if err := ctx.Err(); err != nil {
		return DownloadResult{}, err
	}
	report(len(imageURLs), "done", "download complete")

	log.Printf("asset download complete: site=%s files=%d bytes=%d dir=%s", summary.Site, len(files), totalBytes, chapterDir)
	return DownloadResult{
		OutputDir: chapterDir,
		Files:     files,
		Bytes:     totalBytes,
	}, nil
}

func downloadWorkerCount(imageCount int) int {
	if imageCount <= 0 {
		return 0
	}
	if imageCount < maxDownloadWorkers {
		return imageCount
	}
	return maxDownloadWorkers
}

func downloadOneImage(ctx context.Context, rawURL, outputDir string, index int, usedNamesMu *sync.Mutex, usedNames map[string]int) (string, int64, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", 0, fmt.Errorf("image url is empty")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", 0, fmt.Errorf("parse image url %q: %w", rawURL, err)
	}

	resp, err := downloadImageResponse(ctx, rawURL, parsed)
	if err != nil {
		return "", 0, err
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
	if needsHitomiReferer(parsed) {
		baseName = fmt.Sprintf("%04d", index)
	}

	usedNamesMu.Lock()
	filename := uniqueDownloadFilename(baseName, ext, index, usedNames)
	usedNamesMu.Unlock()
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

func downloadImageResponse(ctx context.Context, rawURL string, parsed *url.URL) (*http.Response, error) {
	attempts := defaultImageDownloadAttempts
	hitomiCDN := needsHitomiReferer(parsed)
	if hitomiCDN {
		attempts = hitomiImageDownloadAttempts
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create image request %q: %w", rawURL, err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		if hitomiCDN {
			req.Header.Set("Referer", "https://hitomi.la/")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("download image %q: %w", rawURL, err)
			if attempt < attempts {
				sleepBeforeImageRetry(ctx, attempt)
				continue
			}
			return nil, lastErr
		}
		if !retryableImageStatus(resp.StatusCode) || attempt >= attempts {
			return resp, nil
		}
		lastErr = fmt.Errorf("download image %q: unexpected status %s", rawURL, resp.Status)
		_ = resp.Body.Close()
		sleepBeforeImageRetry(ctx, attempt)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("download image %q failed", rawURL)
	}
	return nil, lastErr
}

func retryableImageStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func sleepBeforeImageRetry(ctx context.Context, attempt int) {
	delay := time.Duration(250*attempt) * time.Millisecond
	if delay > 3*time.Second {
		delay = 3 * time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func needsHitomiReferer(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return strings.Contains(host, "gold-usergeneratedcontent.net") || strings.Contains(host, "hitomi.la")
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
