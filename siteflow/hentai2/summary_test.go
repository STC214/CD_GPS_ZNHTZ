package hentai2

import "testing"

func TestParseSummaryPage(t *testing.T) {
	html := `<h1 class="my-4 title"><span class="text-secondary">[Kororin Suttonton (Yukari Onigiri)] </span><span class="text-dark">Kento-kun wa otona no omocha ni kyoumi ga atta dake de mesu ochi suru tsumori wa nakatta youdesu</span><span class="text-secondary"> [Digital]</span></h1>
<li><span class="alert-link">Pages: </span> 36                        </li>
<a href="/read/1092723.html" class="btn btn-success btn-lg"><svg></svg> Read Online</a>`

	page, err := ParseSummaryPage("https://hentai2.example/gallery/1092723.html", html)
	if err != nil {
		t.Fatalf("ParseSummaryPage() error = %v", err)
	}
	if page.PageCount != 36 {
		t.Fatalf("PageCount = %d, want 36", page.PageCount)
	}
	if page.ReaderURL != "https://hentai2.example/read/1092723.html" {
		t.Fatalf("ReaderURL = %q", page.ReaderURL)
	}
	wantTitle := "[Kororin Suttonton (Yukari Onigiri)] Kento-kun wa otona no omocha ni kyoumi ga atta dake de mesu ochi suru tsumori wa nakatta youdesu [Digital]"
	if page.Title != wantTitle {
		t.Fatalf("Title = %q", page.Title)
	}
}
