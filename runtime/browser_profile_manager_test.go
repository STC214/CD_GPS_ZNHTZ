package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserProfileManagerPreparesAndCleansTaskProfile(t *testing.T) {
	workspace := t.TempDir()
	projectMother := filepath.Join(workspace, "runtime", "browser-profiles", "baseline-userdata")
	if err := os.MkdirAll(filepath.Join(projectMother, "default"), 0o755); err != nil {
		t.Fatalf("create project mother dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectMother, "prefs.js"), []byte("user_pref(\"browser.startup.homepage\", \"about:blank\");"), 0o644); err != nil {
		t.Fatalf("write project mother file: %v", err)
	}

	appData := filepath.Join(workspace, "AppData", "Roaming")
	localAppData := filepath.Join(workspace, "AppData", "Local")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", localAppData)
	if err := os.MkdirAll(filepath.Join(appData, "Mozilla", "Firefox"), 0o755); err != nil {
		t.Fatalf("create firefox appdata dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appData, "Mozilla", "Firefox", "profiles.ini"), []byte(`
[Profile0]
Name=default
IsRelative=1
Path=Profiles/it9mcsht.default
Default=1
`), 0o644); err != nil {
		t.Fatalf("write profiles.ini: %v", err)
	}
	firefoxSource := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "it9mcsht.default")
	if err := os.MkdirAll(firefoxSource, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	t.Setenv("COMIC_FIREFOX_PROFILE_SOURCE_DIR", firefoxSource)

	paths := NewBrowserPaths(workspace)
	manager := NewBrowserProfileManager(workspace)

	mother, err := manager.SourceProfileDir(BrowserTypeFirefox)
	if err != nil {
		t.Fatalf("SourceProfileDir() error = %v", err)
	}
	if mother != filepath.Clean(firefoxSource) {
		t.Fatalf("SourceProfileDir() = %q, want %q", mother, firefoxSource)
	}
	if got := manager.MotherProfileDir(BrowserTypeFirefox); got != filepath.Clean(projectMother) {
		t.Fatalf("MotherProfileDir() = %q, want %q", got, projectMother)
	}

	taskProfile, err := manager.PrepareTaskProfile(BrowserTypeFirefox, "worker-a", "123")
	if err != nil {
		t.Fatalf("PrepareTaskProfile() error = %v", err)
	}

	wantRoot := filepath.Join(paths.BrowserTasksRoot, string(BrowserTypeFirefox), "worker-a", "task-123")
	if taskProfile.RootDir != wantRoot {
		t.Fatalf("taskProfile.RootDir = %q, want %q", taskProfile.RootDir, wantRoot)
	}
	if taskProfile.MotherProfileDir != projectMother {
		t.Fatalf("taskProfile.MotherProfileDir = %q, want %q", taskProfile.MotherProfileDir, projectMother)
	}
	if got, err := os.ReadFile(filepath.Join(taskProfile.OriginalUserData, "prefs.js")); err != nil || string(got) != "user_pref(\"browser.startup.homepage\", \"about:blank\");" {
		t.Fatalf("original prefs read = %q, err = %v", string(got), err)
	}
	if got, err := os.ReadFile(filepath.Join(taskProfile.ContentUserData, "default", "marker.txt")); err == nil {
		t.Fatalf("content marker unexpectedly exists = %q", string(got))
	}

	if err := os.WriteFile(filepath.Join(taskProfile.OriginalUserData, "default", "marker.txt"), []byte("mother"), 0o644); err != nil {
		t.Fatalf("write task original marker: %v", err)
	}
	if err := copyDir(taskProfile.OriginalUserData, taskProfile.ContentUserData); err != nil {
		t.Fatalf("copy task original to content: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(taskProfile.ContentUserData, "default", "marker.txt")); err != nil || string(got) != "mother" {
		t.Fatalf("content marker read = %q, err = %v", string(got), err)
	}

	match, ok := paths.ResolveExact(taskProfile.OriginalUserData)
	if !ok {
		t.Fatal("ResolveExact(task original) = false, want true")
	}
	if match.Kind != BrowserPathKindTaskOriginal || match.Browser != BrowserTypeFirefox {
		t.Fatalf("ResolveExact(task original) = %+v, want firefox task original", match)
	}

	if err := manager.CleanupTaskProfile(BrowserTypeFirefox, "worker-a", "123"); err != nil {
		t.Fatalf("CleanupTaskProfile() error = %v", err)
	}
	if _, err := os.Stat(wantRoot); !os.IsNotExist(err) {
		t.Fatalf("task root exists after cleanup, stat err = %v", err)
	}
}

func TestBrowserProfileManagerPreparesTaskProfileFromSource(t *testing.T) {
	workspace := t.TempDir()
	appData := filepath.Join(workspace, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))

	sourceDir := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "q6nkoa5l.default-default-1")
	if err := os.MkdirAll(filepath.Join(sourceDir, "default"), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "prefs.js"), []byte("source"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	manager := NewBrowserProfileManager(workspace)
	taskProfile, err := manager.PrepareTaskProfileFromSource(BrowserTypeFirefox, sourceDir, "worker-b", "456")
	if err != nil {
		t.Fatalf("PrepareTaskProfileFromSource() error = %v", err)
	}
	if taskProfile.MotherProfileDir != filepath.Clean(sourceDir) {
		t.Fatalf("taskProfile.MotherProfileDir = %q, want %q", taskProfile.MotherProfileDir, sourceDir)
	}
	got, err := os.ReadFile(filepath.Join(taskProfile.OriginalUserData, "prefs.js"))
	if err != nil {
		t.Fatalf("read copied prefs.js: %v", err)
	}
	if string(got) != "source" {
		t.Fatalf("copied prefs.js = %q, want %q", string(got), "source")
	}
}

func TestBrowserProfileManagerPreparesPlaywrightProfileFromSource(t *testing.T) {
	workspace := t.TempDir()
	appData := filepath.Join(workspace, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))

	sourceDir := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "jo2klram.default-release")
	if err := os.MkdirAll(filepath.Join(sourceDir, "extensions"), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	t.Setenv("COMIC_FIREFOX_PROFILE_SOURCE_DIR", sourceDir)
	if err := os.WriteFile(filepath.Join(sourceDir, "prefs.js"), []byte("source-playwright"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	manager := NewBrowserProfileManager(workspace)
	profile, err := manager.PreparePlaywrightProfileFromSource(BrowserTypeFirefox, sourceDir)
	if err != nil {
		t.Fatalf("PreparePlaywrightProfileFromSource() error = %v", err)
	}
	if profile.SourceProfileDir != filepath.Clean(sourceDir) {
		t.Fatalf("profile.SourceProfileDir = %q, want %q", profile.SourceProfileDir, sourceDir)
	}
	if got, err := os.ReadFile(filepath.Join(profile.RootDir, "prefs.js")); err != nil || string(got) != "source-playwright" {
		t.Fatalf("copied prefs.js = %q, err = %v", string(got), err)
	}
	if err := manager.CleanupPlaywrightProfile(profile); err != nil {
		t.Fatalf("CleanupPlaywrightProfile() error = %v", err)
	}
	if _, err := os.Stat(profile.RootDir); !os.IsNotExist(err) {
		t.Fatalf("playwright profile root exists after cleanup, stat err = %v", err)
	}
}

func TestBrowserProfileManagerRefreshProjectProfileFromSource(t *testing.T) {
	workspace := t.TempDir()
	appData := filepath.Join(workspace, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))

	sourceDir := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "jo2klram.default-release")
	if err := os.MkdirAll(filepath.Join(sourceDir, "extensions"), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	t.Setenv("COMIC_FIREFOX_PROFILE_SOURCE_DIR", sourceDir)
	if err := os.WriteFile(filepath.Join(sourceDir, "prefs.js"), []byte("source-refresh"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	manager := NewBrowserProfileManager(workspace)
	result, err := manager.RefreshProjectProfileFromSource(BrowserTypeFirefox, sourceDir)
	if err != nil {
		t.Fatalf("RefreshProjectProfileFromSource() error = %v", err)
	}
	wantTarget := filepath.Join(workspace, "runtime", "browser-profiles", "baseline-userdata")
	if result.TargetProfileDir != wantTarget {
		t.Fatalf("result.TargetProfileDir = %q, want %q", result.TargetProfileDir, wantTarget)
	}
	got, err := os.ReadFile(filepath.Join(wantTarget, "prefs.js"))
	if err != nil {
		t.Fatalf("read refreshed prefs.js: %v", err)
	}
	if string(got) != "source-refresh" {
		t.Fatalf("refreshed prefs.js = %q, want %q", string(got), "source-refresh")
	}
}

func TestCopyDirCopiesFirefoxLockFiles(t *testing.T) {
	workspace := t.TempDir()
	sourceDir := filepath.Join(workspace, "source")
	dstDir := filepath.Join(workspace, "dst")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "prefs.js"), []byte("prefs"), 0o644); err != nil {
		t.Fatalf("write prefs.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "parent.lock"), []byte("locked"), 0o644); err != nil {
		t.Fatalf("write parent.lock: %v", err)
	}

	if err := copyDir(sourceDir, dstDir); err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}
	gotLock, err := os.ReadFile(filepath.Join(dstDir, "parent.lock"))
	if err != nil {
		t.Fatalf("read copied parent.lock: %v", err)
	}
	if string(gotLock) != "locked" {
		t.Fatalf("copied parent.lock = %q, want %q", string(gotLock), "locked")
	}
	got, err := os.ReadFile(filepath.Join(dstDir, "prefs.js"))
	if err != nil {
		t.Fatalf("read copied prefs.js: %v", err)
	}
	if string(got) != "prefs" {
		t.Fatalf("copied prefs.js = %q, want %q", string(got), "prefs")
	}
}
