package runtime

import (
	"path/filepath"
	"strings"
)

const defaultFirefoxExecutablePath = `C:\Program Files\Mozilla Firefox\firefox.exe`

// BrowserType identifies the project-managed browser family.
type BrowserType string

const (
	// BrowserTypeFirefox identifies the bundled Firefox runtime.
	BrowserTypeFirefox BrowserType = "firefox"
)

// BrowserPathKind identifies a precise project-managed browser path.
type BrowserPathKind string

const (
	BrowserPathKindRuntimeRoot        BrowserPathKind = "runtime-root"
	BrowserPathKindBrowserProfiles    BrowserPathKind = "browser-profiles-root"
	BrowserPathKindBrowserBaseline    BrowserPathKind = "browser-baseline-userdata"
	BrowserPathKindBrowserTasks       BrowserPathKind = "browser-tasks-root"
	BrowserPathKindBrowserVerify      BrowserPathKind = "browser-verification-root"
	BrowserPathKindFirefoxUserDataDir BrowserPathKind = "firefox-user-data-dir"
	BrowserPathKindFirefoxStealthJS   BrowserPathKind = "firefox-stealth-script"
	BrowserPathKindTaskRoot           BrowserPathKind = "task-root"
	BrowserPathKindTaskOriginal       BrowserPathKind = "task-original-userdata"
	BrowserPathKindTaskContent        BrowserPathKind = "task-content-userdata"
	BrowserPathKindTaskVerify         BrowserPathKind = "task-verify-userdata"
)

// BrowserPaths is the exact browser/runtime directory layout used by this project.
type BrowserPaths struct {
	WorkspaceRoot        string
	RuntimeRoot          string
	BrowserProfilesRoot  string
	BrowserTasksRoot     string
	BrowserVerifyRoot    string
	BrowserBaselineRoot  string
	ThumbnailsRoot       string
	FirefoxUserDataDir   string
	FirefoxStealthScript string
}

// BrowserPathMatch describes one precisely recognized project path.
type BrowserPathMatch struct {
	Kind     BrowserPathKind `json:"kind"`
	Browser  BrowserType     `json:"browser,omitempty"`
	Path     string          `json:"path"`
	WorkerID string          `json:"workerId,omitempty"`
	TaskID   string          `json:"taskId,omitempty"`
}

// NewBrowserPaths builds the project browser path layout from a workspace root.
func NewBrowserPaths(workspaceRoot string) BrowserPaths {
	workspaceRoot = normalizeRoot(workspaceRoot)
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	runtimeRoot := ResolveRuntimeRoot(workspaceRoot)
	return BrowserPaths{
		WorkspaceRoot:        workspaceRoot,
		RuntimeRoot:          runtimeRoot,
		BrowserProfilesRoot:  filepath.Join(runtimeRoot, "browser-profiles"),
		BrowserTasksRoot:     filepath.Join(runtimeRoot, "browser-profiles", "tasks"),
		BrowserVerifyRoot:    filepath.Join(runtimeRoot, "browser-profiles", "verification"),
		BrowserBaselineRoot:  filepath.Join(runtimeRoot, "browser-profiles", "baseline-userdata"),
		ThumbnailsRoot:       filepath.Join(runtimeRoot, "thumbnails"),
		FirefoxUserDataDir:   filepath.Join(runtimeRoot, "browser-profiles", "baseline-userdata"),
		FirefoxStealthScript: filepath.Join(runtimeRoot, "firefox_stealth.js"),
	}
}

// DefaultFirefoxUserDataDir returns the exact project path used for Firefox userdata.
func DefaultFirefoxUserDataDir(runtimeRoot string) string {
	return filepath.Join(normalizeRoot(runtimeRoot), "browser-profiles", "baseline-userdata")
}

// DefaultFirefoxInstallDir returns the project-default Firefox install directory.
func DefaultFirefoxInstallDir(runtimeRoot string) string {
	return filepath.Join(normalizeRoot(runtimeRoot), "playwright-browsers", "firefox")
}

// DefaultPlaywrightDriverDir returns the project-default Playwright driver directory.
func DefaultPlaywrightDriverDir(runtimeRoot string) string {
	return filepath.Join(normalizeRoot(runtimeRoot), "playwright-browsers", "driver")
}

// DefaultFirefoxExecutablePath returns the exact bundled Firefox executable path.
func DefaultFirefoxExecutablePath(runtimeRoot string) string {
	_ = runtimeRoot
	return defaultFirefoxExecutablePath
}

// DefaultFirefoxStealthScript returns the exact Firefox stealth script path.
func DefaultFirefoxStealthScript(runtimeRoot string) string {
	return filepath.Join(normalizeRoot(runtimeRoot), "firefox_stealth.js")
}

// TaskRoot returns the exact task-scoped profile root for one worker/task pair.
func (p BrowserPaths) TaskRoot(workerID, taskID string) string {
	return filepath.Join(p.BrowserTasksRoot, normalizeRoot(workerID), "task-"+normalizeRoot(taskID))
}

// TaskBrowserRoot returns the exact task-scoped profile root for one browser family, worker, and task.
func (p BrowserPaths) TaskBrowserRoot(browserType BrowserType, workerID, taskID string) string {
	return filepath.Join(p.BrowserTasksRoot, string(browserType), normalizeRoot(workerID), "task-"+normalizeRoot(taskID))
}

// TaskOriginalUserData returns the exact task-scoped original-userdata path.
func (p BrowserPaths) TaskOriginalUserData(workerID, taskID string) string {
	return filepath.Join(p.TaskRoot(workerID, taskID), "original-userdata")
}

// TaskBrowserOriginalUserData returns the exact task-scoped original-userdata path for one browser family.
func (p BrowserPaths) TaskBrowserOriginalUserData(browserType BrowserType, workerID, taskID string) string {
	return filepath.Join(p.TaskBrowserRoot(browserType, workerID, taskID), "original-userdata")
}

// TaskContentUserData returns the exact task-scoped content profile path.
func (p BrowserPaths) TaskContentUserData(workerID, taskID string) string {
	return filepath.Join(p.TaskRoot(workerID, taskID), "content")
}

// TaskBrowserContentUserData returns the exact task-scoped content profile path for one browser family.
func (p BrowserPaths) TaskBrowserContentUserData(browserType BrowserType, workerID, taskID string) string {
	return filepath.Join(p.TaskBrowserRoot(browserType, workerID, taskID), "content")
}

// TaskVerifyUserData returns the exact task-scoped verify profile path.
func (p BrowserPaths) TaskVerifyUserData(workerID, taskID string) string {
	return filepath.Join(p.TaskRoot(workerID, taskID), "verify")
}

// TaskThumbnailPath returns the exact task-scoped thumbnail path.
func (p BrowserPaths) TaskThumbnailPath(taskID string) string {
	return filepath.Join(p.ThumbnailsRoot, "task-"+normalizeRoot(taskID), "thumb.jpg")
}

// TaskBrowserVerifyUserData returns the exact task-scoped verify profile path for one browser family.
func (p BrowserPaths) TaskBrowserVerifyUserData(browserType BrowserType, workerID, taskID string) string {
	return filepath.Join(p.TaskBrowserRoot(browserType, workerID, taskID), "verify")
}

// BrowserUserDataDir returns the exact managed userdata directory for a browser family.
func (p BrowserPaths) BrowserUserDataDir(browserType BrowserType) string {
	switch browserType {
	case BrowserTypeFirefox:
		return p.FirefoxUserDataDir
	default:
		return ""
	}
}

// BrowserExecutablePath returns the exact managed browser executable for a browser family.
func (p BrowserPaths) BrowserExecutablePath(browserType BrowserType) string {
	switch browserType {
	case BrowserTypeFirefox:
		return defaultFirefoxExecutablePath
	default:
		return ""
	}
}

// BrowserStealthScript returns the exact managed stealth script for a browser family.
func (p BrowserPaths) BrowserStealthScript(browserType BrowserType) string {
	switch browserType {
	case BrowserTypeFirefox:
		return p.FirefoxStealthScript
	default:
		return ""
	}
}

// ResolveExact recognizes only the exact project-managed browser paths.
func (p BrowserPaths) ResolveExact(path string) (BrowserPathMatch, bool) {
	target, ok := canonicalPath(path)
	if !ok {
		return BrowserPathMatch{}, false
	}
	for _, candidate := range p.exactCandidates() {
		if sameCanonicalPath(target, candidate.path) {
			return candidate.match, true
		}
	}
	if match, ok := p.matchTaskPath(target); ok {
		return match, true
	}
	return BrowserPathMatch{}, false
}

func (p BrowserPaths) exactCandidates() []struct {
	path  string
	match BrowserPathMatch
} {
	return []struct {
		path  string
		match BrowserPathMatch
	}{
		{path: p.RuntimeRoot, match: BrowserPathMatch{Kind: BrowserPathKindRuntimeRoot, Path: p.RuntimeRoot}},
		{path: p.BrowserProfilesRoot, match: BrowserPathMatch{Kind: BrowserPathKindBrowserProfiles, Path: p.BrowserProfilesRoot}},
		{path: p.BrowserTasksRoot, match: BrowserPathMatch{Kind: BrowserPathKindBrowserTasks, Path: p.BrowserTasksRoot}},
		{path: p.BrowserVerifyRoot, match: BrowserPathMatch{Kind: BrowserPathKindBrowserVerify, Path: p.BrowserVerifyRoot}},
		{path: p.BrowserBaselineRoot, match: BrowserPathMatch{Kind: BrowserPathKindBrowserBaseline, Browser: BrowserTypeFirefox, Path: p.BrowserBaselineRoot}},
		{path: p.FirefoxUserDataDir, match: BrowserPathMatch{Kind: BrowserPathKindFirefoxUserDataDir, Browser: BrowserTypeFirefox, Path: p.FirefoxUserDataDir}},
		{path: p.FirefoxStealthScript, match: BrowserPathMatch{Kind: BrowserPathKindFirefoxStealthJS, Browser: BrowserTypeFirefox, Path: p.FirefoxStealthScript}},
	}
}

func (p BrowserPaths) matchTaskPath(path string) (BrowserPathMatch, bool) {
	base, ok := canonicalPath(p.BrowserTasksRoot)
	if !ok {
		return BrowserPathMatch{}, false
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return BrowserPathMatch{}, false
	}
	parts := splitPath(rel)
	switch len(parts) {
	case 3:
		workerID := parts[0]
		taskSegment := parts[1]
		if !strings.HasPrefix(taskSegment, "task-") {
			return BrowserPathMatch{}, false
		}
		taskID := strings.TrimPrefix(taskSegment, "task-")
		switch parts[2] {
		case "original-userdata":
			return BrowserPathMatch{
				Kind:     BrowserPathKindTaskOriginal,
				Path:     path,
				WorkerID: workerID,
				TaskID:   taskID,
			}, true
		case "content":
			return BrowserPathMatch{
				Kind:     BrowserPathKindTaskContent,
				Path:     path,
				WorkerID: workerID,
				TaskID:   taskID,
			}, true
		case "verify":
			return BrowserPathMatch{
				Kind:     BrowserPathKindTaskVerify,
				Path:     path,
				WorkerID: workerID,
				TaskID:   taskID,
			}, true
		default:
			return BrowserPathMatch{}, false
		}
	case 4:
		browserType := BrowserType(parts[0])
		workerID := parts[1]
		taskSegment := parts[2]
		if !strings.HasPrefix(taskSegment, "task-") {
			return BrowserPathMatch{}, false
		}
		taskID := strings.TrimPrefix(taskSegment, "task-")
		switch parts[3] {
		case "original-userdata":
			return BrowserPathMatch{
				Kind:     BrowserPathKindTaskOriginal,
				Browser:  browserType,
				Path:     path,
				WorkerID: workerID,
				TaskID:   taskID,
			}, true
		case "content":
			return BrowserPathMatch{
				Kind:     BrowserPathKindTaskContent,
				Browser:  browserType,
				Path:     path,
				WorkerID: workerID,
				TaskID:   taskID,
			}, true
		case "verify":
			return BrowserPathMatch{
				Kind:     BrowserPathKindTaskVerify,
				Browser:  browserType,
				Path:     path,
				WorkerID: workerID,
				TaskID:   taskID,
			}, true
		default:
			return BrowserPathMatch{}, false
		}
	default:
		return BrowserPathMatch{}, false
	}
}

func normalizeRoot(value string) string {
	return filepath.Clean(strings.TrimSpace(value))
}

func canonicalPath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return filepath.Clean(abs), true
}

func sameCanonicalPath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftAbs, err := filepath.Abs(left)
	if err != nil {
		return false
	}
	rightAbs, err := filepath.Abs(right)
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
}

func splitPath(value string) []string {
	value = filepath.Clean(value)
	if value == "." {
		return nil
	}
	return strings.Split(value, string(filepath.Separator))
}
