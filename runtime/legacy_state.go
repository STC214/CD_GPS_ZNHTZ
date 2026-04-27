package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LegacyComicDownloaderState is the persisted UI snapshot format produced by the old project.
type LegacyComicDownloaderState struct {
	Version     int                         `json:"version"`
	SavedAt     time.Time                   `json:"savedAt"`
	NextTaskID  int                         `json:"nextTaskId"`
	Concurrency int                         `json:"concurrency"`
	Tasks       []LegacyComicDownloaderTask `json:"tasks"`
}

// LegacyComicDownloaderTask is one entry in the old persisted UI snapshot.
type LegacyComicDownloaderTask struct {
	ID            int       `json:"id"`
	URL           string    `json:"url"`
	Title         string    `json:"title"`
	DownloadRoot  string    `json:"downloadRoot"`
	OutputDir     string    `json:"outputDir"`
	Headless      bool      `json:"headless"`
	HTTPOnly      bool      `json:"httpOnly"`
	ThumbnailPath string    `json:"thumbnailPath"`
	Worker        string    `json:"worker"`
	WorkerSource  string    `json:"workerSource"`
	State         string    `json:"state"`
	Detail        string    `json:"detail"`
	Percent       float64   `json:"percent"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// LoadLegacyComicDownloaderState reads the old UI snapshot file from disk.
func LoadLegacyComicDownloaderState(path string) (LegacyComicDownloaderState, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return LegacyComicDownloaderState{}, fmt.Errorf("legacy state path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return LegacyComicDownloaderState{}, err
	}
	var state LegacyComicDownloaderState
	if err := json.Unmarshal(data, &state); err != nil {
		return LegacyComicDownloaderState{}, fmt.Errorf("unmarshal legacy comic downloader state %q: %w", path, err)
	}
	return state, nil
}

// SaveLegacyComicDownloaderState writes the old UI snapshot file to disk.
func SaveLegacyComicDownloaderState(path string, state LegacyComicDownloaderState) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return fmt.Errorf("legacy state path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create legacy state dir %q: %w", filepath.Dir(path), err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	state.SavedAt = time.Now().UTC()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal legacy comic downloader state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write legacy comic downloader state %q: %w", path, err)
	}
	return nil
}

// ResolveLegacyComicDownloaderStatePath resolves the legacy history file path.
func ResolveLegacyComicDownloaderStatePath(workspaceRoot string) string {
	if override := strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_STATE_PATH")); override != "" {
		return filepath.Clean(override)
	}
	return DefaultComicDownloaderStatePath(ResolveRuntimeRoot(workspaceRoot))
}

// DefaultComicDownloaderStatePath returns the default legacy history file path under runtime.
func DefaultComicDownloaderStatePath(runtimeRoot string) string {
	return filepath.Join(normalizeRoot(runtimeRoot), "comic_downloader_state.json")
}
