# Nyahentai Flow Rules

This document records the current `nyahentai` page-processing rules.

The public UI runs this flow through Firefox.

## Inputs

- User entry URL is already the reader page.
- Nyahentai has no separate summary page in this implementation.
- The task title and page count are resolved from the reader page.

## Reader Page Rules

1. The title is taken from:
   - `#post-data > h1`
2. Target images are collected inside:
   - `#post-comic`
3. The gallery id is extracted from the reader URL.
4. The id must be a numeric sequence of at least 6 digits.
5. Target image URLs must contain that id.

## Lazy Loading Rules

1. The browser scrolls the `#post-comic` region until lazy images are loaded.
2. Large galleries may take longer; slow lazy loading is expected for image-heavy pages.
3. A target image is considered usable when its width and height metadata indicate a real page image.
4. The final page count is the number of filtered images found after lazy loading.

## Image Selection Rules

1. Image URLs can come from DOM attributes such as `src`, `srcset`, `data-src`, or lazy-loading variants.
2. Browser image records are preferred when available because they include actual loaded image metadata.
3. Thumbnail, preview, cover, icon, and sprite-like URLs are excluded when they do not match full-page criteria.
4. Final URLs are sorted by page index when the URL path reveals one.

## Failure Rules

- If the URL does not contain a usable id, fail early.
- If `#post-comic` yields no target images, fail instead of reporting a successful empty task.
- If browser image records are unavailable, fall back to parsed reader HTML.

## Notes

- The frontend task result stores `site=nyahentai`.
- Downloading and thumbnail generation are shared with other sites through `siteflow/assets`.
