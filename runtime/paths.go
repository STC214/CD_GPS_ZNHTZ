package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Paths describes the runtime directory layout used by the UI.
type Paths struct {
	Root            string
	BrowserRoot     string
	BrowserTasks    string
	BrowserVerify   string
	BrowserBaseline string
	TasksRoot       string
	LogsRoot        string
	OutputRoot      string
	ThumbnailsRoot  string
}

// NewPaths builds the default runtime layout under workspaceRoot.
func NewPaths(workspaceRoot string) Paths {
	root := ResolveRuntimeRoot(workspaceRoot)
	return NewPathsFromRuntimeRoot(root)
}

// NewPathsFromRuntimeRoot builds the runtime layout from an already resolved runtime root.
func NewPathsFromRuntimeRoot(runtimeRoot string) Paths {
	root := normalizeRoot(runtimeRoot)
	if root == "" {
		root = "."
	}
	return Paths{
		Root:            root,
		BrowserRoot:     filepath.Join(root, "browser-profiles"),
		BrowserTasks:    filepath.Join(root, "browser-profiles", "tasks"),
		BrowserVerify:   filepath.Join(root, "browser-profiles", "verification"),
		BrowserBaseline: filepath.Join(root, "browser-profiles", "baseline-userdata"),
		TasksRoot:       filepath.Join(root, "tasks"),
		LogsRoot:        filepath.Join(root, "logs"),
		OutputRoot:      filepath.Join(root, "output"),
		ThumbnailsRoot:  filepath.Join(root, "thumbnails"),
	}
}

// Ensure creates the runtime directories used by the current UI package.
func (p Paths) Ensure() error {
	for _, dir := range []string{
		p.Root,
		p.BrowserRoot,
		p.BrowserTasks,
		p.BrowserVerify,
		p.BrowserBaseline,
		p.TasksRoot,
		p.LogsRoot,
		p.OutputRoot,
		p.ThumbnailsRoot,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create runtime dir %q: %w", dir, err)
		}
	}
	return nil
}

// TaskReportPath returns the path to one task report file.
func (p Paths) TaskReportPath(taskID string) string {
	return filepath.Join(p.TasksRoot, "task-"+taskID, "report.json")
}

// TaskThumbnailPath returns the path to the task thumbnail image.
func (p Paths) TaskThumbnailPath(taskID string) string {
	return filepath.Join(p.ThumbnailsRoot, "task-"+taskID, "thumb.jpg")
}

// TaskLogPath returns the path to the task human-readable log file.
func (p Paths) TaskLogPath(taskID string) string {
	return filepath.Join(p.LogsRoot, "task-"+sanitizePathPart(taskID)+".log")
}

// DefaultFirefoxProfileDir returns the exact project-owned Firefox profile directory used by tasks.
func DefaultFirefoxProfileDir() string {
	return filepath.Join(ResolveRuntimeRoot("."), "browser-profiles", "baseline-userdata")
}

// DefaultFirefoxProfileSourceDir returns an optional fixed Firefox source profile.
// Empty means the resolver should use COMIC_FIREFOX_PROFILE_SOURCE_DIR or profiles.ini discovery.
func DefaultFirefoxProfileSourceDir() string {
	return ""
}

// DefaultDownloadDir returns the default persistent download directory for the workspace.
func DefaultDownloadDir(workspaceRoot string) string {
	if override := strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_DOWNLOAD_DIR")); override != "" {
		if cleaned, ok := cleanDownloadDirCandidate(override); ok {
			return cleaned
		}
	}
	return filepath.Join(ResolveRuntimeRoot(workspaceRoot), "output")
}

// ResolveRuntimeRoot resolves the persistent runtime root for the current workspace.
func ResolveRuntimeRoot(workspaceRoot string) string {
	if override := strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_RUNTIME_ROOT")); override != "" {
		return filepath.Clean(override)
	}
	return filepath.Join(normalizeRoot(workspaceRoot), "runtime")
}

func cleanDownloadDirCandidate(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	cleaned := filepath.Clean(raw)
	if strings.ContainsAny(cleaned, "<>\"|?*") {
		return "", false
	}
	for _, r := range cleaned {
		if r < 32 {
			return "", false
		}
	}
	if idx := strings.IndexByte(cleaned, ':'); idx >= 0 && idx != 1 {
		return "", false
	}
	return cleaned, true
}

// RelativizePath returns path relative to base when path is inside base.
// If the path cannot be safely relativized, the cleaned original path is returned.
func RelativizePath(base, path string) string {
	base = filepath.Clean(strings.TrimSpace(base))
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if base == "" || !filepath.IsAbs(base) || !filepath.IsAbs(path) {
		return path
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.Clean(rel)
}

// ResolvePath returns an absolute path when path is relative to base.
// Absolute paths are returned cleaned as-is.
func ResolvePath(base, path string) string {
	base = filepath.Clean(strings.TrimSpace(base))
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) || base == "" {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}

func sanitizePathPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "task"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			fallthrough
		case r >= 'A' && r <= 'Z':
			fallthrough
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
