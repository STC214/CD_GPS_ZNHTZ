# Zeri Flow Rules

This document records the current `zeri` page-processing rules so the browser flow can be rebuilt cleanly.

The public UI currently runs this flow through Firefox.

## Inputs

- User entry URL is the summary/article page.
- The workflow must resolve the real reader page before collecting any downloadable images.

## Summary page rules

1. The summary page contains the manga title.
2. The summary page contains the expected page count in text like:
   - `Length : 14 pages`
3. The number in `Length : N pages` is the expected image count for the full reader flow.
4. The summary page contains two `div.row` blocks that point to the reader page.
5. The reader page URL must be resolved from those `div.row` blocks first.
6. The summary page must not be used as the final download source.
7. The summary page must not download cover art as chapter content.

## Reader page rules

1. The reader page must be entered before collecting any final images.
2. After the reader page is open, click the `100%` button once using a real mouse click on the `#image_width1 button` control.
3. After the `100%` click, repeatedly scroll the page up and down until the target reader images have all finished loading.
4. Each reader pagination page must also follow the same order:
   - open the page
   - click `100%`
   - scroll until the target images are all loaded
   - then re-collect the page images
5. The reader pager exists in both:
   - `#page_num1`
   - `#page_num2`
6. Pagination links from both pager blocks must be merged and deduplicated.

## Image selection rules

1. Reader images must be taken only from the reader page content, not the summary/list page.
2. A target image URL must contain two shared numeric signatures.
3. Each shared signature must be at least 6 digits long.
4. All valid reader image URLs must share the same pair of numeric signatures.
5. If cover images or list thumbnails do not match those shared signatures, they must be excluded.
6. Downloaded files should keep the original image filename when possible, with a suffix only for collisions.
7. The shared asset layer must accept common comic formats such as JPG, PNG, GIF, WebP, and AVIF when creating thumbnails.
8. After filtering, the final collected image count should match the expected page count from the summary page.
9. The downloaded chapter should also generate a portrait-style JPG thumbnail through `siteflow/assets` so the UI task card can show the manga cover.
10. The task card should prefer the manga title from the resolved content, not the site host name.

## Navigation rules

1. The browser should start from the user-provided summary page URL.
2. Resolve the reader page URL from the summary page DOM.
3. Enter the reader page in the same browser session when possible.
4. Do not open a second browser path for a different mode during the same task.
5. Do not leave the task stuck on the summary page when the reader page is available.

## Failure rules

- If the reader page cannot be resolved, fail early and log the summary page HTML snapshot.
- If the reader page opens but the image count is lower than the summary page `Length : N pages`, treat it as a mismatch.
- If the final collected images are empty, do not fall back to list-page cover images.
- If a challenge page or verification page appears instead of the reader content, stop and mark the task as blocked.

## Notes

- Keep the summary page and reader page responsibilities separate.
- Prefer explicit logging of:
  - resolved reader URL
  - expected page count
  - pagination URLs
  - collected image count
- Downloading and thumbnail generation are shared with other sites through `siteflow/assets`.
- The download pipeline preserves source filenames when possible, and the thumbnail pipeline produces JPG thumbnails.
