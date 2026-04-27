//go:build windows

package main

import (
	"testing"

	"comic_downloader_go_playwright_stealth/tasks"
	"comic_downloader_go_playwright_stealth/ui"
)

func TestTaskSiteKey(t *testing.T) {
	tests := []struct {
		name string
		item ui.TodoItem
		want string
	}{
		{
			name: "result site",
			item: ui.TodoItem{Result: tasks.BrowserRunResult{Site: "nyahentai"}},
			want: "nyahentai",
		},
		{
			name: "request host zeri",
			item: ui.TodoItem{Request: tasks.BrowserLaunchRequest{URL: "https://www.zerobywzip.com/read/123"}},
			want: "zeri",
		},
		{
			name: "request host hentai2",
			item: ui.TodoItem{Request: tasks.BrowserLaunchRequest{URL: "https://hentai2.example/title"}},
			want: "hentai2",
		},
		{
			name: "request host hentaiaz",
			item: ui.TodoItem{Request: tasks.BrowserLaunchRequest{URL: "https://hentaiaz.com/title"}},
			want: "hentaiaz",
		},
		{
			name: "request host hitomi",
			item: ui.TodoItem{Request: tasks.BrowserLaunchRequest{URL: "https://hitomi.la/galleries/123.html"}},
			want: "hitomi",
		},
		{
			name: "unknown",
			item: ui.TodoItem{Request: tasks.BrowserLaunchRequest{URL: "https://example.com/title"}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taskSiteKey(tt.item); got != tt.want {
				t.Fatalf("taskSiteKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
