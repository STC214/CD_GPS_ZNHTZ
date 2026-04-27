//go:build playwright

package main

import (
	"flag"
	"fmt"
	"log"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
	"comic_downloader_go_playwright_stealth/tasks"
)

const defaultFirefoxUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:149.0) Gecko/20100101 Firefox/149.0"

func main() {
	url := flag.String("url", "", "url to open")
	browserType := flag.String("browser-type", "firefox", "browser type to use: firefox")
	runtimeRoot := flag.String("runtime-root", "runtime", "runtime root")
	browserPath := flag.String("browser-path", "", "browser executable path")
	browsersPath := flag.String("browsers-path", "", "browser install root")
	driverDir := flag.String("driver-dir", "", "playwright driver directory")
	profileDir := flag.String("profile-dir", "", "selected browser profile directory")
	userDataDir := flag.String("user-data-dir", "", "browser profile directory")
	downloadRoot := flag.String("download-root", "", "download root directory")
	outputDir := flag.String("output-dir", "", "output directory for downloaded files")
	launchTimeoutMS := flag.Int("launch-timeout-ms", 120000, "maximum time in milliseconds to wait for browser launch")
	userAgent := flag.String("user-agent", defaultFirefoxUserAgent, "browser user agent")
	locale := flag.String("locale", "en-US", "browser locale for stable testing")
	timezoneID := flag.String("timezone-id", "Asia/Shanghai", "browser timezone for stable testing")
	viewportWidth := flag.Int("viewport-width", 1365, "browser viewport width")
	viewportHeight := flag.Int("viewport-height", 768, "browser viewport height")
	firefoxUserPrefsJSON := flag.String("firefox-user-prefs-json", "", "Firefox user prefs as JSON")
	workerID := flag.String("worker-id", "", "worker id for task-scoped profile copy")
	taskID := flag.String("task-id", "", "task id for task-scoped profile copy")
	headless := flag.Bool("headless", true, "run browser headless")
	adblock := flag.Bool("adblock", true, "enable adblock flag in middleware")
	keepOpen := flag.Bool("keep-open", false, "keep the browser open until the window is closed")
	flag.Parse()
	if cleanup, logPath, err := projectruntime.InitProcessLogging(*runtimeRoot, "task-probe"); err != nil {
		log.Fatalf("init task-probe logging: %v", err)
	} else {
		log.Printf("task-probe logging: %s", logPath)
		defer func() {
			if err := cleanup(); err != nil {
				log.Printf("close task-probe log: %v", err)
			}
		}()
	}

	req := tasks.BrowserLaunchRequest{
		URL:                  *url,
		BrowserType:          *browserType,
		Headless:             *headless,
		LaunchTimeoutMS:      *launchTimeoutMS,
		RuntimeRoot:          *runtimeRoot,
		BrowserPath:          *browserPath,
		BrowserInstallDir:    *browsersPath,
		DriverDir:            *driverDir,
		ProfileDir:           *profileDir,
		UserDataDir:          *userDataDir,
		DownloadRoot:         *downloadRoot,
		OutputDir:            *outputDir,
		UserAgent:            *userAgent,
		Locale:               *locale,
		TimezoneID:           *timezoneID,
		ViewportWidth:        *viewportWidth,
		ViewportHeight:       *viewportHeight,
		FirefoxUserPrefsJSON: *firefoxUserPrefsJSON,
		Adblock:              *adblock,
		WorkerID:             *workerID,
		TaskID:               *taskID,
		KeepOpen:             *keepOpen,
	}

	result, err := tasks.RunBrowserRequest(req)
	if err != nil {
		log.Fatalf("run browser request: %v", err)
	}
	log.Printf("task-probe title: %s", result.Title)
	fmt.Println(result.Title)
	if result.PlaywrightProfileDir != "" {
		log.Printf("task-probe profile: %s", result.PlaywrightProfileDir)
		fmt.Println(result.PlaywrightProfileDir)
	}
}
