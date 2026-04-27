package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
)

// BrowserMenuState is the top-bar browser selection surface used by the frontend.
type BrowserMenuState struct {
	SelectedBrowser       string
	FirefoxExecutablePath string
	FirefoxInstallRoot    string
	PlaywrightDriverDir   string
}

// PlaywrightRootResolution describes paths found under one Playwright browser root.
type PlaywrightRootResolution struct {
	Root                string
	FirefoxExecutable   string
	PlaywrightDriverDir string
}

// DefaultBrowserMenuState returns the current browser selection defaults for the frontend.
func DefaultBrowserMenuState() BrowserMenuState {
	paths := projectruntime.NewPaths(".")
	return BrowserMenuState{
		SelectedBrowser:       "firefox",
		FirefoxExecutablePath: projectruntime.DefaultFirefoxExecutablePath(paths.Root),
		FirefoxInstallRoot:    projectruntime.DefaultFirefoxInstallDir(paths.Root),
		PlaywrightDriverDir:   projectruntime.DefaultPlaywrightDriverDir(paths.Root),
	}
}

// WithFirefoxExecutablePath updates the Firefox executable path shown in the top menu.
func (m BrowserMenuState) WithFirefoxExecutablePath(executablePath string) BrowserMenuState {
	m.FirefoxExecutablePath = executablePath
	return m
}

// WithFirefoxInstallRoot updates the Playwright Firefox install directory.
func (m BrowserMenuState) WithFirefoxInstallRoot(installRoot string) BrowserMenuState {
	m.FirefoxInstallRoot = installRoot
	return m
}

// WithPlaywrightDriverDir updates the Playwright driver directory.
func (m BrowserMenuState) WithPlaywrightDriverDir(driverDir string) BrowserMenuState {
	m.PlaywrightDriverDir = driverDir
	return m
}

// WithSelectedBrowser updates the browser picked in the top menu.
func (m BrowserMenuState) WithSelectedBrowser(browser string) BrowserMenuState {
	m.SelectedBrowser = browser
	return m
}

// ResolvePlaywrightRoot finds the Firefox executable and Playwright driver under one root directory.
func ResolvePlaywrightRoot(root string) (PlaywrightRootResolution, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return PlaywrightRootResolution{}, fmt.Errorf("Playwright root is empty")
	}
	info, err := os.Stat(root)
	if err != nil {
		return PlaywrightRootResolution{}, fmt.Errorf("Playwright root %q: %w", root, err)
	}
	if !info.IsDir() {
		return PlaywrightRootResolution{}, fmt.Errorf("Playwright root %q is not a directory", root)
	}
	firefox, firefoxErr := projectruntime.ResolveInstalledBrowserExecutable(root, projectruntime.BrowserTypeFirefox)
	driverDir, driverErr := resolvePlaywrightDriverDir(root)
	if firefoxErr != nil || driverErr != nil {
		if firefoxErr != nil && driverErr != nil {
			return PlaywrightRootResolution{}, fmt.Errorf("resolve Playwright root %q: %v; %v", root, firefoxErr, driverErr)
		}
		if firefoxErr != nil {
			return PlaywrightRootResolution{}, firefoxErr
		}
		return PlaywrightRootResolution{}, driverErr
	}
	return PlaywrightRootResolution{
		Root:                root,
		FirefoxExecutable:   firefox,
		PlaywrightDriverDir: driverDir,
	}, nil
}

// WithPlaywrightRoot applies a resolved Playwright browser root to all browser settings.
func (m BrowserMenuState) WithPlaywrightRoot(root string) (BrowserMenuState, PlaywrightRootResolution, error) {
	resolved, err := ResolvePlaywrightRoot(root)
	if err != nil {
		return m, PlaywrightRootResolution{}, err
	}
	m.FirefoxInstallRoot = resolved.Root
	m.FirefoxExecutablePath = resolved.FirefoxExecutable
	m.PlaywrightDriverDir = resolved.PlaywrightDriverDir
	return m, resolved, nil
}

// ResolvePlaywrightDriverDir accepts either a Playwright driver directory or its parent root.
func ResolvePlaywrightDriverDir(path string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return "", fmt.Errorf("Playwright driver path is empty")
	}
	if isPlaywrightDriverDir(path) {
		return path, nil
	}
	resolved, err := ResolvePlaywrightRoot(path)
	if err != nil {
		return "", err
	}
	return resolved.PlaywrightDriverDir, nil
}

func resolvePlaywrightDriverDir(root string) (string, error) {
	candidates := []string{root}
	if strings.ToLower(filepath.Base(root)) != "driver" {
		candidates = append([]string{filepath.Join(root, "driver")}, candidates...)
	}
	for _, candidate := range candidates {
		if isPlaywrightDriverDir(candidate) {
			return filepath.Clean(candidate), nil
		}
	}
	var found []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "driver") && isPlaywrightDriverDir(path) {
			found = append(found, filepath.Clean(path))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("scan Playwright driver under %q: %w", root, err)
	}
	if len(found) == 0 {
		return "", fmt.Errorf("Playwright driver directory not found under %q", root)
	}
	return found[0], nil
}

func isPlaywrightDriverDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "node.exe")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "package")); err != nil {
		return false
	}
	return true
}
