package zeri

import "testing"

func TestParseSummaryPageResolvesReaderURL(t *testing.T) {
	html := `
	<html>
		<head><title>my manga - List - Page : 1</title></head>
		<body>
			<div class="row">
				<a href="/index.php?route=comic/view&id=123">reader one</a>
			</div>
			<div class="row">
				<a href="https://zeri-m.top/index.php?route=comic/view&id=123">reader two</a>
			</div>
			<p>Length : 14 pages</p>
		</body>
	</html>`

	page, err := ParseSummaryPage("https://zeri-m.top/index.php?route=comic/list&tag_id=320", html)
	if err != nil {
		t.Fatalf("ParseSummaryPage() error = %v", err)
	}
	if page.PageCount != 14 {
		t.Fatalf("PageCount = %d, want 14", page.PageCount)
	}
	if page.ReaderURL != "https://zeri-m.top/index.php?route=comic/view&id=123" {
		t.Fatalf("ReaderURL = %q", page.ReaderURL)
	}
	if page.RowCount != 2 {
		t.Fatalf("RowCount = %d, want 2", page.RowCount)
	}
	if page.Title == "" {
		t.Fatal("Title is empty")
	}
}
