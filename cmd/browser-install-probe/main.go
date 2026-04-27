package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/ui"
)

func main() {
	defaultTargetRoot := filepath.Join("runtime", "playwright-browsers")
	targetRoot := flag.String("target-root", defaultTargetRoot, "browser install directory")
	flag.Parse()

	workspaceRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("get working directory: %v", err)
	}

	middleware := ui.NewBrowserInstallMiddleware(workspaceRoot)
	start := time.Now()
	result, err := middleware.InstallAllBrowsersWithProgress(*targetRoot, func(progress runtime.BrowserInstallProgress) {
		payload := map[string]any{
			"fraction": progress.Fraction,
			"phase":    progress.Phase,
			"message":  progress.Message,
			"browser":  progress.Browser,
		}
		data, _ := json.Marshal(payload)
		fmt.Println(string(data))
	})
	if err != nil {
		log.Fatalf("install browsers: %v", err)
	}

	summary := map[string]any{
		"workspaceRoot": workspaceRoot,
		"targetRoot":    result.TargetRoot,
		"elapsedMs":     time.Since(start).Milliseconds(),
		"results":       result.Results,
	}
	data, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(data))
}
