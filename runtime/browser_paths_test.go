package runtime

import (
	"path/filepath"
	"testing"
)

func TestBrowserPathsResolveExact(t *testing.T) {
	paths := NewBrowserPaths("F:/Project/comic_downloader_GO_Playwright_stealth")

	match, ok := paths.ResolveExact("F:/Project/comic_downloader_GO_Playwright_stealth/runtime/browser-profiles/baseline-userdata")
	if !ok {
		t.Fatal("ResolveExact(firefox userdata) = false, want true")
	}
	if match.Kind != BrowserPathKindBrowserBaseline {
		t.Fatalf("match.Kind = %q, want %q", match.Kind, BrowserPathKindBrowserBaseline)
	}
	if match.Browser != BrowserTypeFirefox {
		t.Fatalf("match.Browser = %q, want %q", match.Browser, BrowserTypeFirefox)
	}

	if _, ok := paths.ResolveExact("F:/Project/comic_downloader_GO_Playwright_stealth/runtime/browser-profiles/firefox2"); ok {
		t.Fatal("ResolveExact(firefox2) = true, want false")
	}
}

func TestBrowserPathsTaskRoots(t *testing.T) {
	paths := NewBrowserPaths("F:/Project/comic_downloader_GO_Playwright_stealth")
	base := filepath.Clean(filepath.FromSlash("F:/Project/comic_downloader_GO_Playwright_stealth"))

	if got, want := paths.TaskOriginalUserData("worker-a", "123"), filepath.Join(base, "runtime", "browser-profiles", "tasks", "worker-a", "task-123", "original-userdata"); got != want {
		t.Fatalf("TaskOriginalUserData() = %q, want %q", got, want)
	}
	if got, want := paths.TaskContentUserData("worker-a", "123"), filepath.Join(base, "runtime", "browser-profiles", "tasks", "worker-a", "task-123", "content"); got != want {
		t.Fatalf("TaskContentUserData() = %q, want %q", got, want)
	}
	if got, want := paths.TaskVerifyUserData("worker-a", "123"), filepath.Join(base, "runtime", "browser-profiles", "tasks", "worker-a", "task-123", "verify"); got != want {
		t.Fatalf("TaskVerifyUserData() = %q, want %q", got, want)
	}
}
