package tasks

import (
	"time"

	"comic_downloader_go_playwright_stealth/siteflow"
)

// TaskState is the normalized task state string used by the UI.
type TaskState string

const (
	TaskStateQueued              TaskState = "queued"
	TaskStateRouting             TaskState = "routing"
	TaskStatePreparing           TaskState = "preparing"
	TaskStatePrepared            TaskState = "prepared"
	TaskStateRunning             TaskState = "running"
	TaskStatePaused              TaskState = "paused"
	TaskStateWaitingVerification TaskState = "waiting_verification"
	TaskStateVerificationCleared TaskState = "verification_cleared"
	TaskStateCompleted           TaskState = "completed"
	TaskStateFailed              TaskState = "failed"
)

// TaskReport is the persisted task summary consumed by the UI.
type TaskReport struct {
	TaskID             string                       `json:"taskID"`
	Manifest           siteflow.TaskManifestSummary `json:"manifest"`
	State              TaskState                    `json:"state"`
	Verification       string                       `json:"verification"`
	BrowserType        string                       `json:"browserType,omitempty"`
	BrowserPath        string                       `json:"browserPath,omitempty"`
	BrowserMode        string                       `json:"browserMode,omitempty"`
	PageType           string                       `json:"pageType,omitempty"`
	Verified           bool                         `json:"verified,omitempty"`
	VerificationNeeded bool                         `json:"verificationNeeded,omitempty"`
	Blocked            bool                         `json:"blocked,omitempty"`
	MatchedMarker      string                       `json:"matchedMarker,omitempty"`
	Note               string                       `json:"note,omitempty"`
	OutputRoot         string                       `json:"outputRoot"`
	ThumbnailRoot      string                       `json:"thumbnailRoot"`
	ThumbnailPath      string                       `json:"thumbnailPath,omitempty"`
	StatePath          string                       `json:"statePath"`
	ReportPath         string                       `json:"reportPath"`
	LogPath            string                       `json:"logPath"`
	StorageState       string                       `json:"storageState"`
	VerificationState  string                       `json:"verificationState"`
	InitScript         string                       `json:"initScript"`
	CreatedAt          time.Time                    `json:"createdAt"`
	StartedAt          time.Time                    `json:"startedAt"`
	FinishedAt         time.Time                    `json:"finishedAt"`
	LastError          string                       `json:"lastError"`
}
