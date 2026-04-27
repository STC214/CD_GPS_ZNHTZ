package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FrontendWindowPlacement stores the last window placement of the Win32 frontend.
type FrontendWindowPlacement struct {
	Flags                uint32 `json:"flags,omitempty"`
	ShowCmd              uint32 `json:"showCmd,omitempty"`
	MinPositionX         int32  `json:"minPositionX,omitempty"`
	MinPositionY         int32  `json:"minPositionY,omitempty"`
	MaxPositionX         int32  `json:"maxPositionX,omitempty"`
	MaxPositionY         int32  `json:"maxPositionY,omitempty"`
	NormalPositionLeft   int32  `json:"normalPositionLeft,omitempty"`
	NormalPositionTop    int32  `json:"normalPositionTop,omitempty"`
	NormalPositionRight  int32  `json:"normalPositionRight,omitempty"`
	NormalPositionBottom int32  `json:"normalPositionBottom,omitempty"`
}

// FrontendState stores the persistent settings used by the current frontend.
type FrontendState struct {
	Version               int                     `json:"version"`
	SavedAt               time.Time               `json:"savedAt"`
	SelectedBrowser       string                  `json:"selectedBrowser,omitempty"`
	FirefoxExecutablePath string                  `json:"firefoxExecutablePath,omitempty"`
	FirefoxInstallRoot    string                  `json:"firefoxInstallRoot,omitempty"`
	PlaywrightDriverDir   string                  `json:"playwrightDriverDir,omitempty"`
	DownloadDir           string                  `json:"downloadDir,omitempty"`
	Concurrency           int                     `json:"concurrency,omitempty"`
	ProgressRefreshMs     int                     `json:"progressRefreshMs,omitempty"`
	WindowPlacement       FrontendWindowPlacement `json:"windowPlacement,omitempty"`
}

// ResolveFrontendStatePath resolves the frontend settings file path.
func ResolveFrontendStatePath(workspaceRoot string) string {
	if override := strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_FRONTEND_STATE_PATH")); override != "" {
		return filepath.Clean(override)
	}
	return DefaultFrontendStatePath(ResolveRuntimeRoot(workspaceRoot))
}

// DefaultFrontendStatePath returns the default settings path under runtime.
func DefaultFrontendStatePath(runtimeRoot string) string {
	return filepath.Join(normalizeRoot(runtimeRoot), "frontend_state.json")
}

// LoadFrontendState reads the current frontend state snapshot.
func LoadFrontendState(path string) (FrontendState, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return FrontendState{}, fmt.Errorf("frontend state path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return FrontendState{}, err
	}
	var state FrontendState
	if err := json.Unmarshal(data, &state); err != nil {
		return FrontendState{}, fmt.Errorf("unmarshal frontend state %q: %w", path, err)
	}
	return state, nil
}

// SaveFrontendState writes the current frontend state snapshot to disk.
func SaveFrontendState(path string, state FrontendState) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return fmt.Errorf("frontend state path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create frontend state dir %q: %w", filepath.Dir(path), err)
	}
	state.SavedAt = time.Now().UTC()
	if state.Version == 0 {
		state.Version = 1
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal frontend state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write frontend state %q: %w", path, err)
	}
	return nil
}
