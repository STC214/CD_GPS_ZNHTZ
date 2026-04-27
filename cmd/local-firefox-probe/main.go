//go:build playwright

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func main() {
	url := flag.String("url", "https://example.com", "url to open")
	firefoxPath := flag.String("firefox-path", "", "full path to firefox.exe")
	browsersPath := flag.String("browsers-path", "", "root path containing Playwright browsers")
	headless := flag.Bool("headless", false, "run browser headless")
	flag.Parse()

	exe := strings.TrimSpace(*firefoxPath)
	if exe == "" {
		root := strings.TrimSpace(*browsersPath)
		if root == "" {
			root = os.Getenv("PLAYWRIGHT_BROWSERS_PATH")
		}
		exe = findFirefoxExecutable(root)
	}
	if exe == "" {
		log.Fatal("firefox.exe not found; pass --firefox-path or --browsers-path, or set PLAYWRIGHT_BROWSERS_PATH")
	}
	if _, err := os.Stat(exe); err != nil {
		log.Fatalf("firefox executable %q: %v", exe, err)
	}
	if strings.TrimSpace(*url) == "" {
		log.Fatal("missing --url")
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Firefox.Launch(playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(exe),
		Headless:       playwright.Bool(*headless),
	})
	if err != nil {
		log.Fatalf("launch firefox: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("new page: %v", err)
	}

	if _, err := page.Goto(*url); err != nil {
		log.Fatalf("goto %q: %v", *url, err)
	}

	fmt.Println("opened:", *url)
	select {}
}

func findFirefoxExecutable(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	info, err := os.Stat(root)
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		if strings.EqualFold(filepath.Base(root), "firefox.exe") {
			return filepath.Clean(root)
		}
		return ""
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if d != nil && !d.IsDir() && strings.EqualFold(d.Name(), "firefox.exe") {
			found = filepath.Clean(path)
		}
		return nil
	})
	return found
}
