package hitomi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIBaseURL        = "https://ltn.gold-usergeneratedcontent.net"
	defaultCDNDomain         = "gold-usergeneratedcontent.net"
	defaultFormatDir         = "webp"
	maxHitomiTextBytes int64 = 8 << 20
)

var hashPathRe = regexp.MustCompile(`/[0-9a-f]{61}([0-9a-f]{2})([0-9a-f])`)

// Client resolves Hitomi gallery metadata and image URLs.
type Client struct {
	HTTPClient *http.Client
	APIBaseURL string

	mu          sync.Mutex
	lastGGFetch time.Time
	mDefault    int
	mMap        map[int]int
	bPrefix     string
}

// NewClient creates a Hitomi client with the same endpoint family used by Hitomi itself.
func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
		APIBaseURL: defaultAPIBaseURL,
		mMap:       map[int]int{},
	}
}

// GetComic resolves galleryinfo and generates downloadable image URLs.
func (c *Client) GetComic(ctx context.Context, sourceURL string) (Comic, error) {
	if c == nil {
		c = NewClient()
	}
	id, ok := GalleryIDFromURL(sourceURL)
	if !ok {
		return Comic{}, fmt.Errorf("hitomi gallery id not found in url %q", sourceURL)
	}
	info, err := c.GetGalleryInfo(ctx, id)
	if err != nil {
		return Comic{}, err
	}
	imageURLs := make([]string, 0, len(info.Files))
	for _, file := range info.Files {
		imageURL, err := c.ImageURLFromImage(ctx, id, file, defaultFormatDir)
		if err != nil {
			return Comic{}, fmt.Errorf("build image url for gallery %d file %q: %w", id, file.Name, err)
		}
		imageURLs = append(imageURLs, imageURL)
	}
	title := strings.TrimSpace(info.Title)
	if title == "" {
		title = strings.TrimSpace(info.JapaneseTitle)
	}
	if title == "" {
		title = fmt.Sprintf("hitomi-%d", id)
	}
	return Comic{
		ID:        id,
		Title:     title,
		PageCount: len(imageURLs),
		Files:     info.Files,
		ImageURLs: imageURLs,
		SourceURL: sourceURL,
	}, nil
}

// GetGalleryInfo downloads and parses /galleries/{id}.js.
func (c *Client) GetGalleryInfo(ctx context.Context, id int) (GalleryInfo, error) {
	if id <= 0 {
		return GalleryInfo{}, fmt.Errorf("hitomi gallery id is empty")
	}
	body, err := c.getText(ctx, fmt.Sprintf("%s/galleries/%d.js", c.apiBaseURL(), id))
	if err != nil {
		return GalleryInfo{}, fmt.Errorf("get hitomi galleryinfo %d: %w", id, err)
	}
	jsonText := strings.TrimSpace(body)
	jsonText = strings.TrimPrefix(jsonText, "var galleryinfo = ")
	jsonText = strings.TrimSpace(strings.TrimSuffix(jsonText, ";"))
	var info GalleryInfo
	if err := json.Unmarshal([]byte(jsonText), &info); err != nil {
		return GalleryInfo{}, fmt.Errorf("parse hitomi galleryinfo %d: %w", id, err)
	}
	if info.ID == 0 {
		info.ID = id
	}
	if len(info.Files) == 0 {
		return GalleryInfo{}, fmt.Errorf("hitomi gallery %d has no files", id)
	}
	return info, nil
}

// ImageURLFromImage ports Hitomi's hash -> CDN URL logic for webp/avif images.
func (c *Client) ImageURLFromImage(ctx context.Context, galleryID int, image GalleryFile, dir string) (string, error) {
	hash := strings.TrimSpace(image.Hash)
	if hash == "" {
		return "", fmt.Errorf("image hash is empty")
	}
	dir = strings.TrimSpace(dir)
	ext := dir
	if ext == "" {
		ext = extensionFromName(image.Name)
	}
	if ext == "" {
		ext = defaultFormatDir
	}
	path, err := c.fullPathFromHash(ctx, hash)
	if err != nil {
		return "", err
	}
	raw := "https://a." + defaultCDNDomain + "/" + path + "." + ext
	return c.urlFromURL(ctx, raw, "", dir)
}

func (c *Client) fullPathFromHash(ctx context.Context, hash string) (string, error) {
	b, err := c.b(ctx)
	if err != nil {
		return "", err
	}
	s, err := hashShard(hash)
	if err != nil {
		return "", err
	}
	return b + s + "/" + hash, nil
}

func (c *Client) urlFromURL(ctx context.Context, raw, base, dir string) (string, error) {
	subdomain, err := c.subdomainFromURL(ctx, raw, base, dir)
	if err != nil {
		return "", err
	}
	if subdomain == "" {
		return raw, nil
	}
	replaced := regexp.MustCompile(`//..?\.(?:gold-usergeneratedcontent\.net|hitomi\.la)/`).
		ReplaceAllString(raw, "//"+subdomain+"."+defaultCDNDomain+"/")
	return replaced, nil
}

func (c *Client) subdomainFromURL(ctx context.Context, raw, base, dir string) (string, error) {
	retval := ""
	if base == "" {
		switch dir {
		case "webp":
			retval = "w"
		case "avif":
			retval = "a"
		}
	}
	match := hashPathRe.FindStringSubmatch(raw)
	if len(match) < 3 {
		return retval, nil
	}
	g, err := strconv.ParseInt(match[2]+match[1], 16, 32)
	if err != nil {
		return "", err
	}
	m, err := c.m(ctx, int(g))
	if err != nil {
		return "", err
	}
	if base == "" {
		return retval + strconv.Itoa(1+m), nil
	}
	return string(rune('a'+m)) + base, nil
}

func (c *Client) m(ctx context.Context, g int) (int, error) {
	if err := c.refreshGG(ctx); err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if value, ok := c.mMap[g]; ok {
		return value, nil
	}
	return c.mDefault, nil
}

func (c *Client) b(ctx context.Context) (string, error) {
	if err := c.refreshGG(ctx); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bPrefix, nil
}

func (c *Client) refreshGG(ctx context.Context) error {
	c.mu.Lock()
	if !c.lastGGFetch.IsZero() && time.Since(c.lastGGFetch) < time.Minute {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	body, err := c.getText(ctx, c.apiBaseURL()+"/gg.js")
	if err != nil {
		return fmt.Errorf("get hitomi gg.js: %w", err)
	}
	mDefault, mMap, bPrefix := parseGG(body)
	c.mu.Lock()
	c.mDefault = mDefault
	c.mMap = mMap
	c.bPrefix = bPrefix
	c.lastGGFetch = time.Now()
	c.mu.Unlock()
	return nil
}

func parseGG(body string) (int, map[int]int, string) {
	mDefault := firstIntSubmatch(regexp.MustCompile(`var o = (\d+)`), body)
	o := firstIntSubmatch(regexp.MustCompile(`o = (\d+); break;`), body)
	mMap := map[int]int{}
	for _, match := range regexp.MustCompile(`case (\d+):`).FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		if value, err := strconv.Atoi(match[1]); err == nil {
			mMap[value] = o
		}
	}
	bPrefix := ""
	if match := regexp.MustCompile(`b: '([^']*)'`).FindStringSubmatch(body); len(match) > 1 {
		bPrefix = match[1]
	}
	return mDefault, mMap, bPrefix
}

func firstIntSubmatch(re *regexp.Regexp, text string) int {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	value, _ := strconv.Atoi(match[1])
	return value
}

func hashShard(hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	if len(hash) < 3 {
		return "", fmt.Errorf("invalid hash %q", hash)
	}
	suffix := hash[len(hash)-3:]
	hexValue := suffix[2:] + suffix[:2]
	n, err := strconv.ParseInt(hexValue, 16, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n, 10), nil
}

func extensionFromName(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx < len(name)-1 {
		return strings.ToLower(name[idx+1:])
	}
	return ""
}

func (c *Client) getText(ctx context.Context, rawURL string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://hitomi.la/")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}
	if resp.ContentLength > maxHitomiTextBytes {
		return "", fmt.Errorf("response too large: %d bytes > %d bytes", resp.ContentLength, maxHitomiTextBytes)
	}
	limited := &io.LimitedReader{R: resp.Body, N: maxHitomiTextBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxHitomiTextBytes {
		return "", fmt.Errorf("response too large: %d bytes > %d bytes", len(data), maxHitomiTextBytes)
	}
	return string(data), nil
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) apiBaseURL() string {
	if c != nil && strings.TrimSpace(c.APIBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(c.APIBaseURL), "/")
	}
	return defaultAPIBaseURL
}
