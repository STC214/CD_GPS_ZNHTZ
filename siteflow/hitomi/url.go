package hitomi

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	galleryPathIDRe = regexp.MustCompile(`(?i)/(?:galleries|reader)/(\d+)(?:\.html)?`)
	trailingIDRe    = regexp.MustCompile(`(?i)[-/](\d{4,})(?:\.html)?/?$`)
	anyIDRe         = regexp.MustCompile(`\d{4,}`)
)

// IsHitomiURL reports whether the URL belongs to Hitomi.
func IsHitomiURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return strings.Contains(host, "hitomi.la")
}

// GalleryIDFromURL extracts a Hitomi gallery id from common Hitomi URLs.
func GalleryIDFromURL(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	parsed, err := url.Parse(raw)
	target := raw
	if err == nil {
		target = parsed.Path
	}
	for _, re := range []*regexp.Regexp{galleryPathIDRe, trailingIDRe} {
		if match := re.FindStringSubmatch(target); len(match) > 1 {
			if id, err := strconv.Atoi(match[1]); err == nil && id > 0 {
				return id, true
			}
		}
	}
	if match := anyIDRe.FindString(target); match != "" {
		if id, err := strconv.Atoi(match); err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

func canonicalGalleryURL(id int) string {
	if id <= 0 {
		return ""
	}
	return fmt.Sprintf("https://hitomi.la/galleries/%d.html", id)
}
