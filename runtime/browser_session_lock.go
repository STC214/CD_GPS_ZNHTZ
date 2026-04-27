package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const browserSessionLockFileName = "browser-session.lock"
const processQueryLimitedInformation = 0x1000

// BrowserSessionLockPath returns the exact lock-file path used to signal an active browser session.
func BrowserSessionLockPath(runtimeRoot string) string {
	return BrowserSessionLockPathScoped(runtimeRoot, "")
}

// BrowserSessionLockPathScoped returns the lock-file path for a scoped browser session.
func BrowserSessionLockPathScoped(runtimeRoot, scope string) string {
	root := strings.TrimSpace(runtimeRoot)
	if root == "" {
		return browserSessionLockFileName
	}
	scope = sanitizeBrowserSessionLockScope(scope)
	if scope == "" {
		return filepath.Join(root, browserSessionLockFileName)
	}
	return filepath.Join(root, "browser-session-"+scope+".lock")
}

// BrowserSessionLocked reports whether an active browser session lock currently exists.
func BrowserSessionLocked(runtimeRoot string) bool {
	return BrowserSessionLockedScoped(runtimeRoot, "")
}

// BrowserSessionLockedScoped reports whether an active scoped browser session lock currently exists.
func BrowserSessionLockedScoped(runtimeRoot, scope string) bool {
	_, err := os.Stat(BrowserSessionLockPathScoped(runtimeRoot, scope))
	return err == nil
}

// AcquireBrowserSessionLock creates the active browser session lock file and returns a release function.
func AcquireBrowserSessionLock(runtimeRoot string) (func() error, error) {
	return AcquireBrowserSessionLockScoped(runtimeRoot, "")
}

// AcquireBrowserSessionLockScoped creates the active scoped browser session lock file and returns a release function.
func AcquireBrowserSessionLockScoped(runtimeRoot, scope string) (func() error, error) {
	lockPath := BrowserSessionLockPathScoped(runtimeRoot, scope)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create browser session lock dir: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = file.WriteString(fmt.Sprintf("pid=%d\nstarted=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano)))
			_ = file.Close()
			return func() error {
				return os.Remove(lockPath)
			}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("create browser session lock %q: %w", lockPath, err)
		}
		stale, staleErr := browserSessionLockIsStale(lockPath)
		if staleErr != nil {
			return nil, staleErr
		}
		if !stale {
			return nil, errors.New("browser session is already running")
		}
		if removeErr := os.Remove(lockPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return nil, fmt.Errorf("remove stale browser session lock %q: %w", lockPath, removeErr)
		}
	}
	return nil, errors.New("browser session is already running")
}

// WaitForBrowserSessionUnlock waits until the active browser session lock disappears.
func WaitForBrowserSessionUnlock(runtimeRoot string, pollInterval time.Duration) error {
	return WaitForBrowserSessionUnlockScoped(runtimeRoot, "", pollInterval)
}

// WaitForBrowserSessionUnlockScoped waits until the active scoped browser session lock disappears.
func WaitForBrowserSessionUnlockScoped(runtimeRoot, scope string, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	lockPath := BrowserSessionLockPathScoped(runtimeRoot, scope)
	for {
		if _, err := os.Stat(lockPath); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("stat browser session lock %q: %w", lockPath, err)
		}
		time.Sleep(pollInterval)
	}
}

// BrowserSessionLockInfo returns a human-readable description of the active lock if present.
func BrowserSessionLockInfo(runtimeRoot string) (string, error) {
	lockPath := BrowserSessionLockPath(runtimeRoot)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func sanitizeBrowserSessionLockScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range scope {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "._-")
}

func browserSessionLockIsStale(lockPath string) (bool, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("read browser session lock %q: %w", lockPath, err)
	}
	pid, err := parseBrowserSessionLockPID(string(data))
	if err != nil {
		return true, nil
	}
	return !processRunning(pid), nil
}

func parseBrowserSessionLockPID(content string) (int, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "pid=") {
			continue
		}
		pidText := strings.TrimSpace(strings.TrimPrefix(line, "pid="))
		if pidText == "" {
			break
		}
		pid, err := strconv.Atoi(pidText)
		if err != nil || pid <= 0 {
			return 0, fmt.Errorf("invalid browser session lock pid %q", pidText)
		}
		return pid, nil
	}
	return 0, errors.New("browser session lock pid not found")
}

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)
	return true
}
