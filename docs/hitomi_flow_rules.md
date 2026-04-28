# Hitomi Flow Rules

This document records the current `hitomi` page-processing rules.

Hitomi is different from the browser-backed sites. It does not require Playwright for parsing.

## Inputs

- User entry URL is a Hitomi gallery, reader, or title URL containing the gallery id.
- Supported examples:
  - `https://hitomi.la/galleries/<id>.html`
  - `https://hitomi.la/reader/<id>.html`
  - `https://hitomi.la/<type>/<slug>-<id>.html`

## Gallery Info Rules

1. Extract the numeric gallery id from the URL.
2. Download:
   - `https://ltn.gold-usergeneratedcontent.net/galleries/<id>.js`
3. Remove the JavaScript prefix:
   - `var galleryinfo = `
4. Parse the remaining JSON as gallery metadata.
5. The title is taken from `title`, falling back to `japanese_title`, then `hitomi-<id>`.
6. The page count is the number of `files` entries.

## Image URL Rules

The Go implementation ports the logic from the local reference checkout under:

```text
references\hitomi-downloader\
```

Rules:

1. Download and cache `gg.js` from the Hitomi CDN.
2. Parse the default `m` value, explicit `case` mappings, and `b` path prefix.
3. For each gallery file hash, compute the shard path from the final hash bytes.
4. Build a WebP image URL using the Hitomi CDN path rules.
5. Rewrite the CDN subdomain using the same `m` calculation used by Hitomi.

The generated URLs look like:

```text
https://wN.gold-usergeneratedcontent.net/<b-prefix>/<shard>/<hash>.webp
```

## Download Rules

1. Hitomi image URLs are passed to `siteflow/assets`.
2. Requests to Hitomi CDN hosts send:
   - `Referer: https://hitomi.la/`
   - a normal browser-like `User-Agent`
3. Hitomi CDN downloads retry on:
   - `429`
   - `500`
   - `502`
   - `503`
   - `504`
   - transient network errors
4. Hitomi CDN retries use up to 20 attempts.
5. Downloaded files are named by page order:
   - `0001.webp`
   - `0002.webp`
   - `0003.webp`

## Task Integration

- Hitomi uses the same task result shape as browser-backed sites.
- `BrowserRunResult.Site` is `hitomi`.
- `ReaderURL` and `ResolvedURL` are set to the canonical gallery URL.
- Downloaded images and thumbnails are handled by `siteflow/assets`.
- The frontend site filter recognizes Hitomi tasks.

## Failure Rules

- If no id can be extracted from the URL, fail early.
- If `galleryinfo` cannot be downloaded or parsed, fail before download.
- If the gallery has no files, fail.
- If all image downloads fail after retry, mark the task failed and write the CDN error to the task report.

## Notes

- Hitomi avoids browser resource creation, which reduces profile and driver failure modes.
- The local reference repository is ignored by git and should remain a reference-only dependency.
