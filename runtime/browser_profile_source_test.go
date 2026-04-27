package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserProfileSourceResolverFirefox(t *testing.T) {
	workspace := t.TempDir()
	appData := filepath.Join(workspace, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))
	sourceDir := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "it9mcsht.default")
	t.Setenv("COMIC_FIREFOX_PROFILE_SOURCE_DIR", sourceDir)

	want := sourceDir
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatalf("create firefox source: %v", err)
	}

	got, err := NewBrowserProfileSourceResolver().ResolveFirefox()
	if err != nil {
		t.Fatalf("ResolveFirefox() error = %v", err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("ResolveFirefox() = %q, want %q", got, want)
	}
}

func TestBrowserProfileSourceResolverFirefoxFallsBackToProfilesINI(t *testing.T) {
	workspace := t.TempDir()
	appData := filepath.Join(workspace, "AppData", "Roaming")
	t.Setenv("APPDATA", appData)
	t.Setenv("LOCALAPPDATA", filepath.Join(workspace, "AppData", "Local"))
	t.Setenv("COMIC_FIREFOX_DISABLE_FIXED_SOURCE", "1")

	if err := os.MkdirAll(filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "it9mcsht.default"), 0o755); err != nil {
		t.Fatalf("create firefox source: %v", err)
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

	got, err := NewBrowserProfileSourceResolver().ResolveFirefox()
	if err != nil {
		t.Fatalf("ResolveFirefox() error = %v", err)
	}
	want := filepath.Join(appData, "Mozilla", "Firefox", "Profiles", "it9mcsht.default")
	if got != filepath.Clean(want) {
		t.Fatalf("ResolveFirefox() = %q, want %q", got, want)
	}
}
