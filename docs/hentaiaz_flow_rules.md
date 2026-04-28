# Hentaiaz Flow Rules

This document records the current `hentaiaz` page-processing rules.

The public UI runs this flow through Firefox.

## Inputs

- User entry URL is the summary/gallery page.
- The workflow resolves the reader page before collecting downloadable images.

## Summary Page Rules

1. The summary title is taken from the single:
   - `span.text-pink`
2. The text inside that tag is the title.
3. The expected page count is read from the text following:
   - `<i class="fa fa-file"></i>`
4. The page-count text has the form `N pages`, where `N` can be 1 to 4 digits.
5. The reader URL is taken from an anchor whose `title` starts with:
   - `List Read`
6. Relative reader URLs are resolved against the summary page URL.

## Reader Page Rules

1. The reader page must be opened before image collection.
2. Target reader images are collected from:
   - `section#image-container`
3. Lazy loading uses the shared browser scroll/wait mechanism.
4. The expected image count comes from the summary page count.

## Image Selection Rules

1. Target image URLs should come from the reader image container.
2. Image filtering follows the Hentai2-style shared-signature rules.
3. If the first image reveals a sequential image pattern, the parser may expand `1` through the expected page count.
4. The final image list is passed to `siteflow/assets` for download and thumbnail generation.

## Failure Rules

- If no reader URL is found, fail early.
- If no target images are found after lazy loading, fail instead of downloading cover art.
- If the expected count and collected count disagree, inspect the reader HTML and sequential expansion rule.

## Notes

- Hentaiaz shares most reader-image behavior with Hentai2 but has its own summary selectors.
- The frontend task result stores `site=hentaiaz`.
