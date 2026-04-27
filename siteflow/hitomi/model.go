package hitomi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GalleryFile is one image entry from Hitomi galleryinfo.
type GalleryFile struct {
	Width   int    `json:"width"`
	Hash    string `json:"hash"`
	HasWebP int    `json:"haswebp"`
	HasAVIF int    `json:"hasavif"`
	HasJXL  int    `json:"hasjxl"`
	Name    string `json:"name"`
	Height  int    `json:"height"`
}

// GalleryInfo is the subset of Hitomi galleryinfo needed by this downloader.
type GalleryInfo struct {
	ID                int           `json:"id"`
	Title             string        `json:"title"`
	JapaneseTitle     string        `json:"japanese_title"`
	Language          string        `json:"language"`
	LanguageLocalName string        `json:"language_localname"`
	Type              string        `json:"type"`
	Date              string        `json:"date"`
	Files             []GalleryFile `json:"files"`
}

// UnmarshalJSON accepts Hitomi gallery ids as either numbers or strings.
func (g *GalleryInfo) UnmarshalJSON(data []byte) error {
	type alias GalleryInfo
	var raw struct {
		alias
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*g = GalleryInfo(raw.alias)
	if len(raw.ID) == 0 {
		return nil
	}
	id, err := parseFlexibleInt(raw.ID)
	if err != nil {
		return fmt.Errorf("parse gallery id: %w", err)
	}
	g.ID = id
	return nil
}

func parseFlexibleInt(raw json.RawMessage) (int, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return 0, nil
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return 0, err
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return 0, nil
	}
	return strconv.Atoi(str)
}

// Comic describes the resolved Hitomi gallery and generated image URLs.
type Comic struct {
	ID        int           `json:"id"`
	Title     string        `json:"title"`
	PageCount int           `json:"pageCount"`
	Files     []GalleryFile `json:"files,omitempty"`
	ImageURLs []string      `json:"imageURLs,omitempty"`
	SourceURL string        `json:"sourceURL"`
}

// ExecutionResult describes the resolved Hitomi download flow.
type ExecutionResult struct {
	Comic           Comic    `json:"comic"`
	CollectedImages []string `json:"collectedImages,omitempty"`
	FinalURL        string   `json:"finalURL"`
	FinalTitle      string   `json:"finalTitle"`
	PageCount       int      `json:"pageCount,omitempty"`
}
