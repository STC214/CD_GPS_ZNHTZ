package zeri

import "testing"

func TestParseReaderPageExtractsPaginationAndImages(t *testing.T) {
	html := `
	<html>
		<head><title>reader - Page : 1</title></head>
		<body>
			<div id="page_num1">
				<a href="/index.php?route=comic/view&id=123&page=2">2</a>
				<a href="/index.php?route=comic/view&id=123&page=3">3</a>
			</div>
			<div id="page_num2">
				<a href="/index.php?route=comic/view&id=123&page=3">3</a>
				<a href="/index.php?route=comic/view&id=123&page=4">4</a>
			</div>
			<div class="reader">
				<button>100%</button>
				<img src="https://img.example.com/123456789012_987654321098_01.jpg">
				<img data-src="https://img.example.com/123456789012_987654321098_02.jpg">
			</div>
		</body>
	</html>`

	page, err := ParseReaderPage("https://zeri-m.top/index.php?route=comic/list&tag_id=320", "https://zeri-m.top/index.php?route=comic/view&id=123", html)
	if err != nil {
		t.Fatalf("ParseReaderPage() error = %v", err)
	}
	if !page.HasZoom100 {
		t.Fatal("HasZoom100 = false, want true")
	}
	if page.Zoom100Clicks == 0 {
		t.Fatal("Zoom100Clicks = 0, want > 0")
	}
	if len(page.PaginationURLs) != 3 {
		t.Fatalf("len(PaginationURLs) = %d, want 3", len(page.PaginationURLs))
	}
	if len(page.ImageURLs) != 2 {
		t.Fatalf("len(ImageURLs) = %d, want 2", len(page.ImageURLs))
	}
	if len(page.FilteredImageURLs) != 2 {
		t.Fatalf("len(FilteredImageURLs) = %d, want 2", len(page.FilteredImageURLs))
	}
	if len(page.SharedSignatures) != 2 {
		t.Fatalf("len(SharedSignatures) = %d, want 2", len(page.SharedSignatures))
	}
	if page.SharedSignatures[0] != "123456789012" {
		t.Fatalf("SharedSignatures[0] = %q", page.SharedSignatures[0])
	}
}

func TestSharedNumericSignatures(t *testing.T) {
	urls := []string{
		"https://img.example.com/123456789012_987654321098_01.jpg",
		"https://img.example.com/123456789012_987654321098_02.jpg",
	}
	sigs := SharedNumericSignatures(urls, 6)
	if len(sigs) != 2 {
		t.Fatalf("len(sigs) = %d, want 2", len(sigs))
	}
}

func TestFilterReaderImageURLs(t *testing.T) {
	urls := []string{
		"https://img.example.com/123456789012_987654321098_01.jpg",
		"https://img.example.com/123456789012_987654321098_02.jpg",
		"https://img.example.com/111111111111_222222222222_03.jpg",
	}
	filtered, sigs := FilterReaderImageURLs(urls, 6)
	if len(sigs) != 2 {
		t.Fatalf("len(sigs) = %d, want 2", len(sigs))
	}
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
}

func TestParseReaderActivationHint(t *testing.T) {
	hint := ParseReaderActivationHint(`<div><button>100%</button><span>100%</span></div>`)
	if !hint.HasZoom100 {
		t.Fatal("HasZoom100 = false, want true")
	}
	if hint.ClickText != "100%" {
		t.Fatalf("ClickText = %q, want 100%%", hint.ClickText)
	}
	if hint.ClickCount == 0 {
		t.Fatal("ClickCount = 0, want > 0")
	}
}
