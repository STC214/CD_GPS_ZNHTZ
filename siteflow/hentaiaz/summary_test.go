package hentaiaz

import "testing"

func TestParseSummaryPage(t *testing.T) {
	html := `<span class="text-pink">いのちだいじに</span>
<div><i class="fa fa-file"></i> 24 pages</div>
<a href="https://x.hentaiaz.com/read/645413.html" class="btn btn-primary" title="List Read いのちだいじに"><i class="fa fa-list-ol"></i> <span>All Images</span></a>`

	page, err := ParseSummaryPage("https://hentaiaz.example/gallery/645413.html", html)
	if err != nil {
		t.Fatalf("ParseSummaryPage() error = %v", err)
	}
	if page.Title != "いのちだいじに" {
		t.Fatalf("Title = %q", page.Title)
	}
	if page.PageCount != 24 {
		t.Fatalf("PageCount = %d, want 24", page.PageCount)
	}
	if page.ReaderURL != "https://x.hentaiaz.com/read/645413.html" {
		t.Fatalf("ReaderURL = %q", page.ReaderURL)
	}
}

func TestParseSummaryPageSupportsHrefAfterTitle(t *testing.T) {
	html := `<span class="text-pink">title</span>
<i class="fa fa-file"></i> 1000 pages
<a title="List Read title" class="btn btn-primary" href="/read/645413.html"><span>All Images</span></a>`

	page, err := ParseSummaryPage("https://x.hentaiaz.com/gallery/645413.html", html)
	if err != nil {
		t.Fatalf("ParseSummaryPage() error = %v", err)
	}
	if page.PageCount != 1000 {
		t.Fatalf("PageCount = %d, want 1000", page.PageCount)
	}
	if page.ReaderURL != "https://x.hentaiaz.com/read/645413.html" {
		t.Fatalf("ReaderURL = %q", page.ReaderURL)
	}
}
