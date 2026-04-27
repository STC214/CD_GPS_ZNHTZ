package zeri

import (
	"errors"
	"strings"
	"testing"
)

type fakeSession struct {
	url       string
	contentFn func() string
	clicks    []string
}

func (f *fakeSession) PageURL() string { return f.url }
func (f *fakeSession) Content() (string, error) {
	if f.contentFn == nil {
		return "", errors.New("no content")
	}
	return f.contentFn(), nil
}
func (f *fakeSession) Goto(url string) error {
	f.url = url
	return nil
}
func (f *fakeSession) ClickText(text string) error {
	f.clicks = append(f.clicks, text)
	if strings.TrimSpace(text) == "" {
		return errors.New("empty text")
	}
	return nil
}

func (f *fakeSession) LoadLazyContent() error                               { return nil }
func (f *fakeSession) LoadLazyContentForCount(expectedImageCount int) error { return nil }

func TestExecuteRunsSummaryToReaderFlow(t *testing.T) {
	summaryHTML := `
	<html><head><title>summary</title></head><body>
		<div class="row"><a href="/reader/123">reader</a></div>
		<p>Length : 2 pages</p>
	</body></html>`
	readerHTML := `
	<html><head><title>reader</title></head><body>
		<div id="page_num1"><a href="/reader/123?page=2">2</a></div>
		<div id="page_num2"><a href="/reader/123?page=3">3</a></div>
		<button>100%</button>
		<img src="https://img.example.com/123456789012_987654321098_01.jpg">
		<img src="https://img.example.com/123456789012_987654321098_02.jpg">
	</body></html>`

	calls := 0
	session := &fakeSession{
		url: "https://zeri-m.top/index.php?route=comic/list&tag_id=320",
		contentFn: func() string {
			calls++
			if calls == 1 {
				return summaryHTML
			}
			return readerHTML
		},
	}

	result, err := Execute(session, session.url)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Summary.ReaderURL == "" {
		t.Fatal("result.Summary.ReaderURL is empty")
	}
	if result.ActivationClicks != 0 {
		t.Fatalf("ActivationClicks = %d, want 0", result.ActivationClicks)
	}
	if result.PaginationActivationClicks != 0 {
		t.Fatalf("PaginationActivationClicks = %d, want 0", result.PaginationActivationClicks)
	}
	if len(result.PaginationPages) != 2 {
		t.Fatalf("len(PaginationPages) = %d, want 2", len(result.PaginationPages))
	}
	if len(result.CollectedImages) != 2 {
		t.Fatalf("len(CollectedImages) = %d, want 2", len(result.CollectedImages))
	}
	if len(session.clicks) != 0 {
		t.Fatalf("clicks = %#v, want no clicks", session.clicks)
	}
	if result.Reader.Title == "" {
		t.Fatal("reader title is empty")
	}
}
