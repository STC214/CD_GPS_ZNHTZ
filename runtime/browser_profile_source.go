package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BrowserProfileSourceResolver resolves the actual user-owned browser profile directories.
type BrowserProfileSourceResolver struct {
	AppData                string
	LocalAppData           string
	FirefoxDir             string
	DisableFixedFirefoxDir bool
}

// NewBrowserProfileSourceResolver builds a resolver from the current process environment.
func NewBrowserProfileSourceResolver() BrowserProfileSourceResolver {
	return BrowserProfileSourceResolver{
		AppData:                strings.TrimSpace(os.Getenv("APPDATA")),
		LocalAppData:           strings.TrimSpace(os.Getenv("LOCALAPPDATA")),
		FirefoxDir:             strings.TrimSpace(os.Getenv("COMIC_FIREFOX_PROFILE_SOURCE_DIR")),
		DisableFixedFirefoxDir: envTruthy("COMIC_FIREFOX_DISABLE_FIXED_SOURCE"),
	}
}

// ResolveFirefox returns the actual Firefox profile directory on the local machine.
func (r BrowserProfileSourceResolver) ResolveFirefox() (string, error) {
	if explicit := strings.TrimSpace(r.FirefoxDir); explicit != "" {
		if info, err := os.Stat(explicit); err == nil && info.IsDir() {
			return filepath.Clean(explicit), nil
		}
		return "", fmt.Errorf("firefox profile %q: %w", explicit, os.ErrNotExist)
	}
	if !r.DisableFixedFirefoxDir {
		if explicit := strings.TrimSpace(DefaultFirefoxProfileSourceDir()); explicit != "" {
			if info, err := os.Stat(explicit); err == nil && info.IsDir() {
				return filepath.Clean(explicit), nil
			}
		}
	}
	profilesRoot := filepath.Join(r.AppData, "Mozilla", "Firefox", "Profiles")
	profilesIni := filepath.Join(r.AppData, "Mozilla", "Firefox", "profiles.ini")
	if info, err := os.Stat(profilesIni); err == nil && !info.IsDir() {
		entries, err := parseFirefoxProfilesINI(profilesIni)
		if err != nil {
			return "", err
		}
		path, err := resolveFirefoxDefaultProfile(entries, profilesRoot)
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("firefox profiles.ini not found at %q", profilesIni)
}

func envTruthy(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

type firefoxProfileEntry struct {
	Path     string
	Default  bool
	Locked   bool
	IsRel    bool
	IniOrder int
}

func parseFirefoxProfilesINI(path string) ([]firefoxProfileEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		entries     []firefoxProfileEntry
		current     *firefoxProfileEntry
		sectionType string
		order       int
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionType = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if strings.HasPrefix(strings.ToLower(sectionType), "profile") {
				entries = append(entries, firefoxProfileEntry{IniOrder: order})
				current = &entries[len(entries)-1]
				order++
			} else {
				current = nil
			}
			continue
		}
		if current == nil {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		switch key {
		case "path":
			current.Path = value
		case "default":
			current.Default = value == "1"
		case "isrelative":
			current.IsRel = value != "0"
		case "locked":
			current.Locked = value == "1"
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func resolveFirefoxDefaultProfile(entries []firefoxProfileEntry, profilesRoot string) (string, error) {
	if len(entries) == 0 {
		return "", errors.New("firefox profiles.ini does not define any profile entries")
	}

	resolve := func(entry firefoxProfileEntry) string {
		if entry.Path == "" {
			return ""
		}
		if filepath.IsAbs(entry.Path) {
			return filepath.Clean(entry.Path)
		}
		if entry.IsRel {
			return filepath.Clean(filepath.Join(filepath.Dir(profilesRoot), entry.Path))
		}
		return filepath.Clean(filepath.Join(profilesRoot, entry.Path))
	}

	defaultEntries := make([]firefoxProfileEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Default {
			defaultEntries = append(defaultEntries, entry)
		}
	}

	if len(defaultEntries) == 0 {
		return "", errors.New("firefox profiles.ini does not mark a default profile")
	}
	if len(defaultEntries) > 1 {
		return "", errors.New("firefox profiles.ini marks multiple default profiles")
	}

	path := resolve(defaultEntries[0])
	if path == "" {
		return "", errors.New("firefox default profile path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat firefox default profile %q: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("firefox default profile %q is not a directory", path)
	}
	return path, nil
}
