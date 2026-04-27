package assets

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestDownloadImagesUsesAdaptiveWorkerLimit(t *testing.T) {
	var mu sync.Mutex
	active := 0
	maxActive := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(40 * time.Millisecond)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("image"))

		mu.Lock()
		active--
		mu.Unlock()
	}))
	defer server.Close()

	urls := make([]string, 12)
	for i := range urls {
		urls[i] = fmt.Sprintf("%s/%02d.jpg", server.URL, i+1)
	}
	var final DownloadProgress
	result, err := DownloadImages(CollectionSummary{Site: "test", Title: "parallel"}, urls, t.TempDir(), func(update DownloadProgress) {
		final = update
	})
	if err != nil {
		t.Fatalf("DownloadImages() error = %v", err)
	}
	if len(result.Files) != len(urls) {
		t.Fatalf("len(Files) = %d, want %d", len(result.Files), len(urls))
	}
	if maxActive <= 1 {
		t.Fatalf("maxActive = %d, want concurrent downloads", maxActive)
	}
	if maxActive > maxDownloadWorkers {
		t.Fatalf("maxActive = %d, want at most %d", maxActive, maxDownloadWorkers)
	}
	if final.Current != len(urls) || final.Total != len(urls) || final.Fraction != 1 {
		t.Fatalf("final progress = %+v, want complete", final)
	}
}

func TestDownloadWorkerCount(t *testing.T) {
	tests := []struct {
		images int
		want   int
	}{
		{images: 0, want: 0},
		{images: 1, want: 1},
		{images: 6, want: 6},
		{images: 7, want: 7},
		{images: 8, want: 7},
	}
	for _, tt := range tests {
		if got := downloadWorkerCount(tt.images); got != tt.want {
			t.Fatalf("downloadWorkerCount(%d) = %d, want %d", tt.images, got, tt.want)
		}
	}
}
