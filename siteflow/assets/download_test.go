package assets

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestDownloadWorkerCountForHitomiTargetImages(t *testing.T) {
	if got := downloadWorkerCount(40); got != 7 {
		t.Fatalf("downloadWorkerCount(40) = %d, want 7", got)
	}
}

func TestWriteLimitedImageFileRejectsOversizedBody(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "oversized.jpg")
	_, err := writeLimitedImageFile(targetPath, strings.NewReader("abcdef"), 5)
	if err == nil {
		t.Fatal("writeLimitedImageFile() error = nil, want oversized error")
	}
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("partial file stat error = %v, want not exist", statErr)
	}
}

func TestSanitizePathPartUsesWindowsDirectoryRules(t *testing.T) {
	got := SanitizePathPart(`A:B/C\D*E?F"G<H>I|J [] 漫画`)
	want := `A_B_C_D_E_F_G_H_I_J [] 漫画`
	if got != want {
		t.Fatalf("SanitizePathPart() = %q, want %q", got, want)
	}
}

func TestSanitizePathPartTruncatesOnlyAfter128Characters(t *testing.T) {
	short := strings.Repeat("a", 128)
	if got := SanitizePathPart(short); got != short {
		t.Fatalf("128 char title changed: len=%d value=%q", len([]rune(got)), got)
	}
	long := strings.Repeat("漫", 129)
	got := SanitizePathPart(long)
	if len([]rune(got)) != 64 {
		t.Fatalf("long title length = %d, want 64", len([]rune(got)))
	}
}
