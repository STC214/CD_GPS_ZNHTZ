package runtime

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ResolveLogRoot resolves the persistent log directory for the current workspace.
func ResolveLogRoot(workspaceRoot string) string {
	if override := strings.TrimSpace(os.Getenv("COMIC_DOWNLOADER_LOG_ROOT")); override != "" {
		return filepath.Clean(override)
	}
	return filepath.Join(ResolveRuntimeRoot(workspaceRoot), "logs")
}

// InitProcessLogging creates a process log file under the persistent log root and
// mirrors log package output to both stderr and the file.
func InitProcessLogging(workspaceRoot, processName string) (func() error, string, error) {
	logRoot := ResolveLogRoot(workspaceRoot)
	if err := os.MkdirAll(logRoot, 0o755); err != nil {
		return nil, "", fmt.Errorf("create log root %q: %w", logRoot, err)
	}
	safeName := sanitizeLogFilePart(processName)
	if safeName == "" {
		safeName = "app"
	}
	fileName := fmt.Sprintf("%s-%s-%d.log", safeName, time.Now().UTC().Format("20060102-150405"), os.Getpid())
	logPath := filepath.Join(logRoot, fileName)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open log file %q: %w", logPath, err)
	}
	log.SetOutput(io.MultiWriter(os.Stderr, file))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Lshortfile)
	log.SetPrefix("")
	log.Printf("logging initialized: %s", logPath)
	return func() error {
		return file.Close()
	}, logPath, nil
}

func sanitizeLogFilePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			b.WriteByte('_')
		default:
			if r < 32 {
				b.WriteByte('_')
			} else {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}
