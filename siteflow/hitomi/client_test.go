package hitomi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGalleryIDFromURL(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{raw: "https://hitomi.la/galleries/1234567.html", want: 1234567},
		{raw: "https://hitomi.la/reader/7654321.html#1", want: 7654321},
		{raw: "https://hitomi.la/manga/example-title-3456789.html", want: 3456789},
	}
	for _, tt := range tests {
		got, ok := GalleryIDFromURL(tt.raw)
		if !ok || got != tt.want {
			t.Fatalf("GalleryIDFromURL(%q) = %d, %v; want %d, true", tt.raw, got, ok, tt.want)
		}
	}
}

func TestHashShard(t *testing.T) {
	got, err := hashShard("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa12f")
	if err != nil {
		t.Fatal(err)
	}
	if got != "3858" {
		t.Fatalf("hashShard() = %q, want 3858", got)
	}
}

func TestParseGG(t *testing.T) {
	body := "var o = 1; switch(g){case 10: case 11: o = 2; break;} b: 'abc/'"
	mDefault, mMap, b := parseGG(body)
	if mDefault != 1 {
		t.Fatalf("mDefault = %d", mDefault)
	}
	if mMap[10] != 2 || mMap[11] != 2 {
		t.Fatalf("mMap = %#v", mMap)
	}
	if b != "abc/" {
		t.Fatalf("b = %q", b)
	}
}

func TestExecuteWithClientResolvesGalleryImageURLs(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa12f"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gg.js":
			_, _ = fmt.Fprint(w, "var o = 1; switch(g){case 3858: o = 2; break;} b: 'abc/'")
		case "/galleries/1234567.js":
			_, _ = fmt.Fprintf(w, `var galleryinfo = {"id":"1234567","title":"Example Hitomi","files":[{"width":720,"height":1040,"hash":%q,"name":"001.jpg","haswebp":1}]};`, hash)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.APIBaseURL = server.URL
	result, err := ExecuteWithClient(context.Background(), client, "https://hitomi.la/galleries/1234567.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalTitle != "Example Hitomi" {
		t.Fatalf("FinalTitle = %q", result.FinalTitle)
	}
	want := "https://w3.gold-usergeneratedcontent.net/abc/3858/" + hash + ".webp"
	if len(result.CollectedImages) != 1 || result.CollectedImages[0] != want {
		t.Fatalf("CollectedImages = %#v, want %q", result.CollectedImages, want)
	}
}

func TestGetTextRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", maxHitomiTextBytes+1))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.getText(context.Background(), server.URL)
	if err == nil {
		t.Fatal("getText() error = nil, want oversized response error")
	}
}
