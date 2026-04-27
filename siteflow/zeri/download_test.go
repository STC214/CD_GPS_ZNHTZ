package zeri

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadImagesWritesFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/img/1.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg-one"))
	})
	mux.HandleFunc("/img/2.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg-two"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dir := t.TempDir()
	summary := SummaryPage{Title: "Zeri 测试标题"}
	result, err := DownloadImages(summary, []string{
		server.URL + "/img/1.jpg",
		server.URL + "/img/2.jpg",
	}, dir, nil)
	if err != nil {
		t.Fatalf("DownloadImages() error = %v", err)
	}
	if result.OutputDir == "" {
		t.Fatal("OutputDir is empty")
	}
	if len(result.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(result.Files))
	}
	for _, path := range result.Files {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stat %q: %v", path, err)
		}
	}
	wantDir := filepath.Join(dir, "Zeri_测试标题")
	if result.OutputDir != wantDir {
		t.Fatalf("OutputDir = %q, want %q", result.OutputDir, wantDir)
	}
}

func TestSelectThumbnailSourcePrefersFirstPageNumber(t *testing.T) {
	got := SelectThumbnailSource([]string{
		`F:\out\11.avif`,
		`F:\out\111.jpg`,
		`F:\out\1.avif`,
		`F:\out\1111.png`,
	})
	want := `F:\out\1.avif`
	if got != want {
		t.Fatalf("SelectThumbnailSource() = %q, want %q", got, want)
	}
}

func TestSelectThumbnailSourceAcceptsZeroPaddedFirstPage(t *testing.T) {
	got := SelectThumbnailSource([]string{
		`F:\out\11.avif`,
		`F:\out\01.jpg`,
		`F:\out\111.png`,
	})
	want := `F:\out\01.jpg`
	if got != want {
		t.Fatalf("SelectThumbnailSource() = %q, want %q", got, want)
	}
}
