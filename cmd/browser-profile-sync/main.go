package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"comic_downloader_go_playwright_stealth/ui"
)

func main() {
	workspaceRoot := flag.String("workspace-root", ".", "workspace root")
	sourceDir := flag.String("source-dir", "", "explicit Firefox source profile directory")
	pollInterval := flag.Duration("poll-interval", 200*time.Millisecond, "browser lock poll interval")
	flag.Parse()

	middleware := ui.NewBrowserProfileMiddleware(*workspaceRoot)
	var (
		result ui.BrowserProfileRefreshResult
		err    error
	)
	if strings.TrimSpace(*sourceDir) != "" {
		result, err = middleware.CloseCurrentBrowserAndCopyFirefoxProfileFromSource(*sourceDir, *pollInterval)
	} else {
		result, err = middleware.CloseCurrentBrowserAndCopyFirefoxProfile(*pollInterval)
	}
	if err != nil {
		log.Fatalf("refresh firefox profile: %v", err)
	}

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("encode refresh result: %v", err)
	}
	fmt.Println(string(encoded))
}
