package siteflow

// TaskManifestSummary is the minimal site manifest shape needed by the UI.
type TaskManifestSummary struct {
	Site       string `json:"site"`
	Title      string `json:"title"`
	PrimaryURL string `json:"primaryURL"`
	AssetCount int    `json:"assetCount"`
	Blocked    bool   `json:"blocked"`
}
