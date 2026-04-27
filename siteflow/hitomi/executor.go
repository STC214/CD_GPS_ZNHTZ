package hitomi

import (
	"context"
	"fmt"
	"log"

	"comic_downloader_go_playwright_stealth/siteflow/zeri"
)

// ExecuteWithProgress resolves a Hitomi gallery without a browser and reports progress.
func ExecuteWithProgress(ctx context.Context, rawURL string, progress zeri.DownloadProgressFunc) (ExecutionResult, error) {
	return ExecuteWithClient(ctx, NewClient(), rawURL, progress)
}

// ExecuteWithClient resolves a Hitomi gallery using the supplied client.
func ExecuteWithClient(ctx context.Context, client *Client, rawURL string, progress zeri.DownloadProgressFunc) (ExecutionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = NewClient()
	}
	log.Printf("hitomi execute start: url=%s", rawURL)
	report := func(current, total int, fraction float64, phase, message string) {
		if progress == nil {
			return
		}
		progress(zeri.DownloadProgress{
			Current:  current,
			Total:    total,
			Fraction: clamp01(fraction),
			Phase:    phase,
			Message:  message,
		})
	}
	report(0, 0, 0.05, "parse", "galleryinfo")
	comic, err := client.GetComic(ctx, rawURL)
	if err != nil {
		return ExecutionResult{}, err
	}
	if len(comic.ImageURLs) == 0 {
		return ExecutionResult{}, fmt.Errorf("hitomi target images not found: url=%s id=%d", rawURL, comic.ID)
	}
	report(len(comic.ImageURLs), len(comic.ImageURLs), 1, "parse", "done")
	log.Printf("hitomi parsed: id=%d title=%q images=%d", comic.ID, comic.Title, len(comic.ImageURLs))
	return ExecutionResult{
		Comic:           comic,
		CollectedImages: comic.ImageURLs,
		FinalURL:        canonicalGalleryURL(comic.ID),
		FinalTitle:      comic.Title,
		PageCount:       len(comic.ImageURLs),
	}, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
