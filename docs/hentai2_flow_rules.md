# Hentai2 Flow Rules

This document records the current `hentai2` page-processing rules.

The public UI runs this flow through Firefox.

## Inputs

- User entry URL is the summary/gallery page.
- The workflow must resolve the real `/read/...html` reader page before collecting downloadable images.

## Summary Page Rules

1. The summary page title is taken from:
   - `<h1 class="my-4 title">...</h1>`
2. Text inside nested spans is merged into one title string.
3. The expected page count is taken from text like:
   - `Pages: 36`
4. The reader URL is taken from a `Read Online` anchor whose href points to `/read/<id>.html`.
5. Relative reader URLs are resolved against the summary page URL.

## Reader Page Rules

1. The reader page must be opened before image collection.
2. Reader images are collected from the `.read1.text-center` content area when that area exists.
3. Lazy loading uses the same browser scroll/wait mechanism as the other reader flows.
4. The expected image count comes from the summary page `Pages: N` value.

## Image Selection Rules

1. Target image URLs must contain `uploads`.
2. Target image URLs must contain at least one numeric signature of five or more digits.
3. Images in the same download task should share the same numeric signature.
4. If the page only exposes the first sequential image, such as:
   - `https://cdn20.hentai2.net/uploads/639493/1.jpg`
   then the parser expands it to `1.jpg` through `N.jpg` using the summary page count.
5. The final image list is passed to `siteflow/assets` for downloading and thumbnail generation.

## Failure Rules

- If the summary page cannot resolve a reader URL, fail early.
- If the reader page yields no target images, fail rather than falling back to cover art.
- If the image count is lower than expected after lazy loading and sequential expansion does not apply, treat the task as incomplete and inspect the reader HTML/rules.

## Notes

- Hentai2 does not currently use the Zeri `100%` zoom-button flow.
- Download output and thumbnails use the shared `siteflow/assets` pipeline.
- The frontend task result stores `site=hentai2`, so the site filter can show completed Hentai2 tasks.
