package hentaiaz

import "testing"

func TestParseReaderPageFiltersReadImages(t *testing.T) {
	html := `<html><head><title>reader</title></head><body>
<img src="https://cdn.example.com/uploads/99999/ad.jpg">
<section id="image-container" class="read1 image-container-2 fit-horizontal full-height">
	<img data-src="/uploads/645413/645413_001.jpg">
	<img data-original="/uploads/645413/645413_002.jpg">
	<img src="/uploads/88888/banner.jpg">
</section>
</body></html>`

	page, err := ParseReaderPage("https://x.hentaiaz.com/read/645413.html", "https://x.hentaiaz.com/read/645413.html", html, 2)
	if err != nil {
		t.Fatalf("ParseReaderPage() error = %v", err)
	}
	if len(page.ImageURLs) != 3 {
		t.Fatalf("len(ImageURLs) = %d, want 3", len(page.ImageURLs))
	}
	if page.SharedSignature != "645413" {
		t.Fatalf("SharedSignature = %q, want 645413", page.SharedSignature)
	}
	if len(page.FilteredImageURLs) != 2 {
		t.Fatalf("len(FilteredImageURLs) = %d, want 2", len(page.FilteredImageURLs))
	}
}

func TestParseReaderPageExpandsSingleSequentialImage(t *testing.T) {
	html := `<section id="image-container" class="read1 image-container-2 fit-horizontal full-height"><img src="https://cdn.hentaiaz.example/uploads/645413/1.jpg"></section>`

	page, err := ParseReaderPage("https://x.hentaiaz.com/read/645413.html", "https://x.hentaiaz.com/read/645413.html", html, 3)
	if err != nil {
		t.Fatalf("ParseReaderPage() error = %v", err)
	}
	if len(page.FilteredImageURLs) != 3 {
		t.Fatalf("len(FilteredImageURLs) = %d, want 3", len(page.FilteredImageURLs))
	}
	if page.FilteredImageURLs[2] != "https://cdn.hentaiaz.example/uploads/645413/3.jpg" {
		t.Fatalf("FilteredImageURLs[2] = %q", page.FilteredImageURLs[2])
	}
}
