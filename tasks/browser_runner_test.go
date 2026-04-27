package tasks

import "testing"

func TestRunBrowserRequestRejectsEmptyURL(t *testing.T) {
	_, err := RunBrowserRequest(BrowserLaunchRequest{})
	if err == nil {
		t.Fatal("RunBrowserRequest() error = nil, want error")
	}
}
