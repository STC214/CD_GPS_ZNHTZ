package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/tasks"
)

func TestImportLegacyComicDownloaderState(t *testing.T) {
	list := NewTodoList()
	existing := list.AddPending(tasks.BrowserLaunchRequest{
		URL:         "https://example.com/b",
		BrowserType: "firefox",
	})
	count := list.ImportLegacyComicDownloaderState(projectruntime.LegacyComicDownloaderState{
		NextTaskID: 42,
		Tasks: []projectruntime.LegacyComicDownloaderTask{
			{
				ID:        41,
				URL:       "https://example.com/a",
				Title:     "A",
				Worker:    "nyahentai",
				State:     "done",
				Percent:   1,
				CreatedAt: time.Unix(100, 0).UTC(),
				UpdatedAt: time.Unix(200, 0).UTC(),
			},
			{
				ID:        40,
				URL:       "https://example.com/b",
				Title:     "B",
				Worker:    "myreadingmanga",
				State:     "queued",
				Percent:   0,
				CreatedAt: time.Unix(50, 0).UTC(),
				UpdatedAt: time.Unix(50, 0).UTC(),
			},
		},
	})

	if count != 1 {
		t.Fatalf("ImportLegacyComicDownloaderState() count = %d, want 1", count)
	}
	items := list.Items()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != existing.ID || items[0].Request.URL != existing.Request.URL {
		t.Fatalf("existing item changed unexpectedly: %#v", items[0])
	}
	if items[1].Status != TodoStatusCompleted {
		t.Fatalf("items[1].Status = %q, want completed", items[1].Status)
	}
	if items[1].ID != "legacy-41" {
		t.Fatalf("legacy ids not preserved: %#v", []string{items[1].ID})
	}
}

func TestExportLegacyComicDownloaderState(t *testing.T) {
	list := NewTodoList()
	list.AddPending(tasks.BrowserLaunchRequest{
		URL:          "https://example.com/a",
		BrowserType:  "firefox",
		DownloadRoot: "F:/downloads",
	})
	item, err := list.RunImmediately(tasks.BrowserLaunchRequest{
		URL:          "https://example.com/b",
		BrowserType:  "firefox",
		DownloadRoot: "F:/downloads",
	}, func(req tasks.BrowserLaunchRequest) (tasks.BrowserRunResult, error) {
		return tasks.BrowserRunResult{
			URL:             req.URL,
			Title:           "Comic B",
			DownloadedDir:   "F:/downloads/comic-b",
			ThumbnailPath:   "F:/thumbs/b.jpg",
			BrowserType:     req.BrowserType,
			ResolvedURL:     req.URL,
			PageType:        "content",
			DownloadedCount: 1,
		}, nil
	})
	if err != nil {
		t.Fatalf("RunImmediately() error = %v", err)
	}

	state := list.ExportLegacyComicDownloaderState(3)
	if state.Concurrency != 3 {
		t.Fatalf("state.Concurrency = %d, want 3", state.Concurrency)
	}
	if len(state.Tasks) != 2 {
		t.Fatalf("len(state.Tasks) = %d, want 2", len(state.Tasks))
	}
	if state.Tasks[1].Title != "Comic B" {
		t.Fatalf("state.Tasks[1].Title = %q, want Comic B", state.Tasks[1].Title)
	}
	if state.Tasks[1].OutputDir != filepath.Clean("F:/downloads/comic-b") {
		t.Fatalf("state.Tasks[1].OutputDir = %q, want F:/downloads/comic-b", state.Tasks[1].OutputDir)
	}
	if state.Tasks[1].ThumbnailPath != filepath.Clean("F:/thumbs/b.jpg") {
		t.Fatalf("state.Tasks[1].ThumbnailPath = %q, want F:/thumbs/b.jpg", state.Tasks[1].ThumbnailPath)
	}
	if state.NextTaskID <= 0 {
		t.Fatalf("state.NextTaskID = %d, want positive", state.NextTaskID)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "comic_downloader_state.json")
	if err := list.SaveLegacyComicDownloaderState(path, 3); err != nil {
		t.Fatalf("SaveLegacyComicDownloaderState() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved state not found: %v", err)
	}
	loaded, err := projectruntime.LoadLegacyComicDownloaderState(path)
	if err != nil {
		t.Fatalf("LoadLegacyComicDownloaderState() error = %v", err)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("loaded tasks = %d, want 2", len(loaded.Tasks))
	}
	if loaded.Tasks[1].Title != "Comic B" {
		t.Fatalf("loaded.Tasks[1].Title = %q, want Comic B", loaded.Tasks[1].Title)
	}
	_ = item
}
