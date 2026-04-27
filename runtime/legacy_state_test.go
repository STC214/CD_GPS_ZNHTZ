package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyComicDownloaderState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comic_downloader_state.json")
	content := `{
		"version": 5,
		"savedAt": "2026-04-19T10:52:34.9498178+08:00",
		"nextTaskId": 316,
		"concurrency": 3,
		"tasks": [
			{
				"id": 315,
				"url": "https://example.com/a",
				"title": "A",
				"downloadRoot": "F:\\Downloads",
				"outputDir": "",
				"headless": true,
				"httpOnly": false,
				"thumbnailPath": "thumb.jpg",
				"worker": "nyahentai",
				"workerSource": "site:nyahentai",
				"state": "queued",
				"detail": "queued",
				"percent": 0,
				"createdAt": "2026-04-19T10:51:45.7846796+08:00",
				"updatedAt": "2026-04-19T10:51:45.7846796+08:00"
			}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}

	state, err := LoadLegacyComicDownloaderState(path)
	if err != nil {
		t.Fatalf("LoadLegacyComicDownloaderState() error = %v", err)
	}
	if state.Version != 5 {
		t.Fatalf("state.Version = %d, want 5", state.Version)
	}
	if state.NextTaskID != 316 {
		t.Fatalf("state.NextTaskID = %d, want 316", state.NextTaskID)
	}
	if len(state.Tasks) != 1 {
		t.Fatalf("len(state.Tasks) = %d, want 1", len(state.Tasks))
	}
	if state.Tasks[0].URL != "https://example.com/a" {
		t.Fatalf("state.Tasks[0].URL = %q", state.Tasks[0].URL)
	}
}
