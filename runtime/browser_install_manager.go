package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// BrowserInstallManager installs Playwright-managed browsers into caller-selected directories.
type BrowserInstallManager struct {
	WorkspaceRoot string
}

// BrowserInstallResult describes one browser installation.
type BrowserInstallResult struct {
	BrowserType     BrowserType `json:"browserType"`
	BrowserName     string      `json:"browserName"`
	TargetRoot      string      `json:"targetRoot"`
	ExecutablePath  string      `json:"executablePath"`
	DriverDirectory string      `json:"driverDirectory,omitempty"`
	Installed       bool        `json:"installed"`
}

// BrowserInstallBatchResult describes installing multiple Playwright browsers into one target root.
type BrowserInstallBatchResult struct {
	TargetRoot string                 `json:"targetRoot"`
	Results    []BrowserInstallResult `json:"results"`
}

// BrowserInstallProgress describes a phase of a multi-browser installation.
type BrowserInstallProgress struct {
	Fraction float64     `json:"fraction"`
	Browser  BrowserType `json:"browserType,omitempty"`
	Phase    string      `json:"phase,omitempty"`
	Message  string      `json:"message,omitempty"`
}

// NewBrowserInstallManager builds a browser install manager rooted at workspaceRoot.
func NewBrowserInstallManager(workspaceRoot string) BrowserInstallManager {
	return BrowserInstallManager{WorkspaceRoot: workspaceRoot}
}

// InstallPlaywrightBrowser downloads a Playwright-managed browser into targetRoot and resolves its executable path.
func (m BrowserInstallManager) InstallPlaywrightBrowser(browserType BrowserType, targetRoot string) (BrowserInstallResult, error) {
	return m.InstallPlaywrightBrowserWithProgress(browserType, targetRoot, nil)
}

// InstallPlaywrightBrowserWithProgress downloads a browser into targetRoot and reports progress.
func (m BrowserInstallManager) InstallPlaywrightBrowserWithProgress(browserType BrowserType, targetRoot string, progress func(BrowserInstallProgress)) (BrowserInstallResult, error) {
	browserName, err := browserInstallName(browserType)
	if err != nil {
		return BrowserInstallResult{}, err
	}
	targetRoot = filepath.Clean(strings.TrimSpace(targetRoot))
	if targetRoot == "" {
		return BrowserInstallResult{}, fmt.Errorf("target root is empty")
	}
	absTarget, err := filepath.Abs(targetRoot)
	if err != nil {
		return BrowserInstallResult{}, fmt.Errorf("resolve target root %q: %w", targetRoot, err)
	}
	absTarget = filepath.Clean(absTarget)
	if err := os.MkdirAll(absTarget, 0o755); err != nil {
		return BrowserInstallResult{}, fmt.Errorf("create target root %q: %w", absTarget, err)
	}
	if progress != nil {
		progress(BrowserInstallProgress{Fraction: 0.02, Browser: browserType, Phase: "prepare", Message: "preparing target directory"})
	}

	driverDir := filepath.Join(absTarget, "driver")
	if err := os.MkdirAll(driverDir, 0o755); err != nil {
		return BrowserInstallResult{}, fmt.Errorf("create driver directory %q: %w", driverDir, err)
	}
	if progress != nil {
		progress(BrowserInstallProgress{Fraction: 0.04, Browser: browserType, Phase: "driver-root", Message: "preparing driver directory"})
	}

	previousBrowsersPath := os.Getenv("PLAYWRIGHT_BROWSERS_PATH")
	if err := os.Setenv("PLAYWRIGHT_BROWSERS_PATH", absTarget); err != nil {
		return BrowserInstallResult{}, fmt.Errorf("set PLAYWRIGHT_BROWSERS_PATH: %w", err)
	}
	defer func() {
		if previousBrowsersPath == "" {
			_ = os.Unsetenv("PLAYWRIGHT_BROWSERS_PATH")
			return
		}
		_ = os.Setenv("PLAYWRIGHT_BROWSERS_PATH", previousBrowsersPath)
	}()

	driver, err := playwright.NewDriver(&playwright.RunOptions{
		DriverDirectory: driverDir,
		Browsers:        []string{browserName},
		Verbose:         true,
	})
	if err != nil {
		return BrowserInstallResult{}, fmt.Errorf("create playwright driver: %w", err)
	}
	if progress != nil {
		progress(BrowserInstallProgress{Fraction: 0.08, Browser: browserType, Phase: "driver", Message: "installing playwright driver"})
	}
	if err := driver.Install(); err != nil {
		return BrowserInstallResult{}, fmt.Errorf("install playwright %s: %w", browserName, err)
	}
	if progress != nil {
		progress(BrowserInstallProgress{Fraction: 0.70, Browser: browserType, Phase: "browser", Message: "installing browser"})
	}

	executablePath, err := findInstalledBrowserExecutable(absTarget, browserType)
	if err != nil {
		return BrowserInstallResult{}, err
	}

	return BrowserInstallResult{
		BrowserType:     browserType,
		BrowserName:     browserName,
		TargetRoot:      absTarget,
		ExecutablePath:  executablePath,
		DriverDirectory: driverDir,
		Installed:       true,
	}, nil
}

// InstallPlaywrightBrowsers installs Firefox into the target root.
func (m BrowserInstallManager) InstallPlaywrightBrowsers(targetRoot string) (BrowserInstallBatchResult, error) {
	return m.InstallPlaywrightBrowsersWithProgress(targetRoot, nil)
}

// InstallPlaywrightBrowsersWithProgress installs Firefox into the target root and reports progress.
func (m BrowserInstallManager) InstallPlaywrightBrowsersWithProgress(targetRoot string, progress func(BrowserInstallProgress)) (BrowserInstallBatchResult, error) {
	firefox, err := m.InstallPlaywrightBrowserWithProgress(BrowserTypeFirefox, targetRoot, func(update BrowserInstallProgress) {
		if progress == nil {
			return
		}
		progress(update)
	})
	if err != nil {
		return BrowserInstallBatchResult{}, err
	}
	if progress != nil {
		progress(BrowserInstallProgress{Fraction: 0.98, Phase: "apply", Message: "applying installed browsers"})
	}
	return BrowserInstallBatchResult{
		TargetRoot: firefox.TargetRoot,
		Results:    []BrowserInstallResult{firefox},
	}, nil
}

func browserInstallName(browserType BrowserType) (string, error) {
	switch browserType {
	case BrowserTypeFirefox:
		return "firefox", nil
	default:
		return "", fmt.Errorf("unsupported browser type %q", browserType)
	}
}

func findInstalledBrowserExecutable(root string, browserType BrowserType) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return "", fmt.Errorf("browser install root is empty")
	}
	candidates := browserExecutableNames(browserType)
	if len(candidates) == 0 {
		return "", fmt.Errorf("unsupported browser type %q", browserType)
	}
	found := make([]string, 0, 4)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(strings.TrimSpace(d.Name()))
		for _, candidate := range candidates {
			if name == candidate {
				found = append(found, filepath.Clean(path))
				break
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("scan installed browser executable under %q: %w", root, err)
	}
	if len(found) == 0 {
		return "", fmt.Errorf("installed %s executable not found under %q", browserType, root)
	}
	sort.Slice(found, func(i, j int) bool { return len(found[i]) < len(found[j]) })
	return found[0], nil
}

// ResolveInstalledBrowserExecutable scans root for the browser executable belonging to browserType.
func ResolveInstalledBrowserExecutable(root string, browserType BrowserType) (string, error) {
	return findInstalledBrowserExecutable(root, browserType)
}

func browserExecutableNames(browserType BrowserType) []string {
	switch browserType {
	case BrowserTypeFirefox:
		return []string{"firefox.exe"}
	default:
		return nil
	}
}
