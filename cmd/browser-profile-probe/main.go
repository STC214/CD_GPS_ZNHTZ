package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	projectruntime "comic_downloader_go_playwright_stealth/runtime"
)

type profileProbeResult struct {
	WorkspaceRoot         string `json:"workspaceRoot"`
	RuntimeRoot           string `json:"runtimeRoot"`
	BrowserType           string `json:"browserType"`
	SelectedProfileDir    string `json:"selectedProfileDir"`
	SelectedProfileExists bool   `json:"selectedProfileExists"`
	ProjectMotherDir      string `json:"projectMotherDir"`
	ProjectMotherExists   bool   `json:"projectMotherExists"`
	SourceDir             string `json:"sourceDir"`
	SourceExists          bool   `json:"sourceExists"`
	SourceError           string `json:"sourceError,omitempty"`
	BrowserPath           string `json:"browserPath"`
	StealthScript         string `json:"stealthScript"`
}

func main() {
	workspaceRoot := flag.String("workspace-root", ".", "workspace root")
	flag.Parse()

	paths := projectruntime.NewBrowserPaths(*workspaceRoot)
	manager := projectruntime.NewBrowserProfileManager(*workspaceRoot)
	browserKind := projectruntime.BrowserTypeFirefox
	sourceDir, err := manager.SourceProfileDir(browserKind)
	if err != nil {
		log.Printf("resolve source dir: %v", err)
	}
	projectMotherDir := manager.MotherProfileDir(browserKind)
	selectedProfileDir := ""
	selectedProfileDir = projectruntime.DefaultFirefoxProfileDir()

	result := profileProbeResult{
		WorkspaceRoot:      paths.WorkspaceRoot,
		RuntimeRoot:        paths.RuntimeRoot,
		BrowserType:        string(browserKind),
		SelectedProfileDir: selectedProfileDir,
		ProjectMotherDir:   projectMotherDir,
		SourceDir:          sourceDir,
		BrowserPath:        projectruntime.DefaultFirefoxExecutablePath(*workspaceRoot),
		StealthScript:      paths.BrowserStealthScript(browserKind),
	}
	if err != nil {
		result.SourceError = err.Error()
	}

	if stat, err := os.Stat(projectMotherDir); err == nil && stat.IsDir() {
		result.ProjectMotherExists = true
	}
	if stat, err := os.Stat(selectedProfileDir); err == nil && stat.IsDir() {
		result.SelectedProfileExists = true
	}
	if stat, err := os.Stat(sourceDir); err == nil && stat.IsDir() {
		result.SourceExists = true
	}

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("encode probe result: %v", err)
	}
	fmt.Println(string(encoded))
}
