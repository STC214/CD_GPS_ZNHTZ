package nyahentai

import (
	"testing"

	"comic_downloader_go_playwright_stealth/browser"
)

func TestParseReaderPageFiltersImagesByReaderID(t *testing.T) {
	html := `<html><body>
<div id="post-data">
	<div>meta</div>
	<h1>Nya Title</h1>
</div>
<img src="https://cdn.example.com/comics/999999/ad.jpg">
<div id="post-comic">
	<div class="nested">
		<img data-src="/images/123456/001.jpg">
	</div>
	<img data-srcset="/images/123456/002.jpg 1x, /images/123456/002-large.jpg 2x">
	<img src="/images/888888/banner.jpg">
</div>
</body></html>`

	page, err := ParseReaderPage("https://nyahentai.example/g/123456/read", "https://nyahentai.example/g/123456/read", html)
	if err != nil {
		t.Fatalf("ParseReaderPage() error = %v", err)
	}
	if page.Title != "Nya Title" {
		t.Fatalf("Title = %q", page.Title)
	}
	if page.SharedSignature != "123456" {
		t.Fatalf("SharedSignature = %q, want 123456", page.SharedSignature)
	}
	if len(page.ImageURLs) != 4 {
		t.Fatalf("len(ImageURLs) = %d, want 4", len(page.ImageURLs))
	}
	if len(page.FilteredImageURLs) != 3 {
		t.Fatalf("len(FilteredImageURLs) = %d, want 3", len(page.FilteredImageURLs))
	}
	if page.FilteredImageURLs[1] != "https://nyahentai.example/images/123456/002.jpg" {
		t.Fatalf("FilteredImageURLs[1] = %q", page.FilteredImageURLs[1])
	}
}

func TestReaderIDFromURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "https://nyahentai.example/g/123456/read", want: "123456"},
		{raw: "https://nyahentai.one/fanzine/re3521313/", want: "3521313"},
		{raw: "https://nyahentai.example/read?id=987654", want: "987654"},
		{raw: "https://nyahentai.example/read/12", want: ""},
	}
	for _, tt := range tests {
		if got := readerIDFromURL(tt.raw); got != tt.want {
			t.Fatalf("readerIDFromURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestCollectReaderImageURLsFromRecordsUsesSizedComicImages(t *testing.T) {
	records := []browser.PageImageRecord{
		{Src: "https://static.example/thumb/3521313/cover.jpg", NaturalWidth: 120, NaturalHeight: 160},
		{Src: "https://i.example/galleries/3521313/2.jpg", NaturalWidth: 900, NaturalHeight: 1200},
		{Src: "https://i.example/galleries/3521313/1.jpg", NaturalWidth: 900, NaturalHeight: 1200},
		{Src: "https://i.example/galleries/999999/1.jpg", NaturalWidth: 900, NaturalHeight: 1200},
	}

	urls := collectReaderImageURLsFromRecords("https://nyahentai.one/fanzine/re3521313/", "https://nyahentai.one/fanzine/re3521313/", records, "")
	if len(urls) != 2 {
		t.Fatalf("len(urls) = %d, want 2: %#v", len(urls), urls)
	}
	if urls[0] != "https://i.example/galleries/3521313/1.jpg" {
		t.Fatalf("urls[0] = %q", urls[0])
	}
	if urls[1] != "https://i.example/galleries/3521313/2.jpg" {
		t.Fatalf("urls[1] = %q", urls[1])
	}
}
