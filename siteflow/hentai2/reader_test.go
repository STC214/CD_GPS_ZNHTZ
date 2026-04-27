package hentai2

import "testing"

func TestParseReaderPageFiltersReadImages(t *testing.T) {
	html := `<html><head><title>reader</title></head><body>
<img src="https://cdn.example.com/uploads/99999/ad.jpg">
<div class="read1 text-center">
	<img data-src="/uploads/1092723/1092723_001.jpg">
	<img data-original="/uploads/1092723/1092723_002.jpg">
	<img src="/uploads/88888/banner.jpg">
</div>
</body></html>`

	page, err := ParseReaderPage("https://hentai2.example/read/1092723.html", "https://hentai2.example/read/1092723.html", html, 2)
	if err != nil {
		t.Fatalf("ParseReaderPage() error = %v", err)
	}
	if len(page.ImageURLs) != 3 {
		t.Fatalf("len(ImageURLs) = %d, want 3", len(page.ImageURLs))
	}
	if page.SharedSignature != "1092723" {
		t.Fatalf("SharedSignature = %q, want 1092723", page.SharedSignature)
	}
	if len(page.FilteredImageURLs) != 2 {
		t.Fatalf("len(FilteredImageURLs) = %d, want 2", len(page.FilteredImageURLs))
	}
}

func TestParseReaderPageExpandsSingleSequentialImage(t *testing.T) {
	html := `<div class="read1 text-center"><img src="https://cdn20.hentai2.net/uploads/639493/1.jpg"></div>`

	page, err := ParseReaderPage("https://hentai2.example/read/1092723.html", "https://hentai2.example/read/1092723.html", html, 3)
	if err != nil {
		t.Fatalf("ParseReaderPage() error = %v", err)
	}
	if len(page.FilteredImageURLs) != 3 {
		t.Fatalf("len(FilteredImageURLs) = %d, want 3", len(page.FilteredImageURLs))
	}
	if page.FilteredImageURLs[2] != "https://cdn20.hentai2.net/uploads/639493/3.jpg" {
		t.Fatalf("FilteredImageURLs[2] = %q", page.FilteredImageURLs[2])
	}
}
