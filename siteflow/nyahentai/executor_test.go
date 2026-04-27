package nyahentai

import (
	"strings"
	"testing"

	"comic_downloader_go_playwright_stealth/browser"
	"comic_downloader_go_playwright_stealth/siteflow/zeri"
)

type fakeSession struct {
	html       string
	lazyHTML   string
	lazyCalled bool
	selector   string
	records    []browser.PageImageRecord
}

func (s *fakeSession) PageURL() string { return "https://nyahentai.example/g/123456/read" }
func (s *fakeSession) Content() (string, error) {
	if s.lazyCalled && strings.TrimSpace(s.lazyHTML) != "" {
		return s.lazyHTML, nil
	}
	return s.html, nil
}
func (s *fakeSession) Goto(string) error                 { return nil }
func (s *fakeSession) ClickText(string) error            { return nil }
func (s *fakeSession) LoadLazyContent() error            { s.lazyCalled = true; return nil }
func (s *fakeSession) LoadLazyContentForCount(int) error { s.lazyCalled = true; return nil }
func (s *fakeSession) LoadLazyContentInSelector(selector string) error {
	s.lazyCalled = true
	s.selector = selector
	return nil
}
func (s *fakeSession) ImageRecords() ([]browser.PageImageRecord, error) { return s.records, nil }

func TestExecuteWithProgressUsesLazyFilteredCount(t *testing.T) {
	initial := `<div id="post-data"><h1>Title</h1></div><div id="post-comic"><img data-src="/images/123456/001.jpg"></div>`
	final := `<div id="post-data"><h1>Title</h1></div><div id="post-comic"><img src="/images/123456/001.jpg"><img src="/images/123456/002.jpg"><img src="/images/999999/ad.jpg"></div>`
	session := &fakeSession{
		html:     initial,
		lazyHTML: final,
		records: []browser.PageImageRecord{
			{Src: "https://i.example/galleries/123456/2.jpg", NaturalWidth: 900, NaturalHeight: 1200},
			{Src: "https://i.example/galleries/123456/1.jpg", NaturalWidth: 900, NaturalHeight: 1200},
		},
	}
	var fractions []float64

	result, err := ExecuteWithProgress(session, "https://nyahentai.example/g/123456/read", func(update zeri.DownloadProgress) {
		fractions = append(fractions, update.Fraction)
	})
	if err != nil {
		t.Fatalf("ExecuteWithProgress() error = %v", err)
	}
	if !session.lazyCalled {
		t.Fatalf("LoadLazyContent was not called")
	}
	if session.selector != "#post-comic" {
		t.Fatalf("selector = %q, want #post-comic", session.selector)
	}
	if result.PageCount != 2 {
		t.Fatalf("PageCount = %d, want 2", result.PageCount)
	}
	if len(result.CollectedImages) != 2 {
		t.Fatalf("len(CollectedImages) = %d, want 2", len(result.CollectedImages))
	}
	if result.CollectedImages[0] != "https://i.example/galleries/123456/1.jpg" {
		t.Fatalf("CollectedImages[0] = %q", result.CollectedImages[0])
	}
	if len(fractions) == 0 || fractions[len(fractions)-1] != 1 {
		t.Fatalf("progress fractions = %+v, want final 1", fractions)
	}
}

func TestExecuteWithProgressFailsWhenNoTargetImages(t *testing.T) {
	html := `<div id="post-data"><h1>Title</h1></div><div id="post-comic"><img src="/images/999999/ad.jpg"></div>`
	session := &fakeSession{html: html, lazyHTML: html}

	_, err := ExecuteWithProgress(session, "https://nyahentai.example/g/123456/read", nil)
	if err == nil {
		t.Fatalf("ExecuteWithProgress() error = nil, want target image error")
	}
	if !strings.Contains(err.Error(), "target images not found") {
		t.Fatalf("ExecuteWithProgress() error = %v", err)
	}
}
