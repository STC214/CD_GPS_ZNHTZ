# Project Audit

Audit date: 2026-04-28

## Scope

This audit summarizes the current repository structure, supported site flows, runtime data layout, frontend/task contract, and the latest safety review. It is a static/local review of the code and documentation. Live browser downloads are intentionally outside the audit path.

## Verification

Command:

```powershell
$env:GOTMPDIR=(Join-Path (Get-Location) '.gotmp')
$env:GOCACHE=(Join-Path (Get-Location) '.gocache')
go test ./...
```

Result: pass.

Covered packages include:

- `browser`
- `runtime`
- `siteflow/assets`
- `siteflow/hentai2`
- `siteflow/zeri`
- `tasks`
- `ui`
- `cmd/win32-frontend`

## Architecture Snapshot

- `cmd/win32-frontend`: Windows desktop UI entry point.
- `cmd/portable-launcher`: single-file launcher that extracts the packaged frontend and points persistent data at `portable-data\`.
- `cmd/task-probe`: Playwright-backed task smoke-test entry point, built with `-tags playwright`.
- `browser`: Playwright session middleware for Firefox, stealth injection, adblock route setup, and lazy-load scrolling.
- `runtime`: runtime paths, fresh browser profile handling, frontend state, logging, and browser install helpers.
- `tasks`: task-level browser request normalization, site dispatch, progress mapping, downloading, thumbnails, and result reporting.
- `siteflow/assets`: shared image downloading, thumbnail source selection, and JPG thumbnail creation.
- `siteflow/zeri`: Zeri URL detection, summary parsing, reader parsing, pagination walking, and image URL collection.
- `siteflow/hentai2`: Hentai2 URL detection, summary parsing, reader parsing, lazy image collection, and sequential image expansion fallback.
- `siteflow/hentaiaz`: Hentaiaz URL detection, summary parsing, reader parsing, lazy image collection, and sequential image expansion fallback.
- `siteflow/nyahentai`: Nyahentai direct reader URL detection, reader-title parsing, lazy image collection, and reader-ID image filtering.
- `siteflow/hitomi`: Hitomi URL detection, gallery metadata parsing, `gg.js` rule parsing, CDN URL generation, and image URL collection without a Playwright browser.
- `netproxy`: proxy normalization and HTTP transport creation for backend downloads.
- `ui`: task list, legacy history, task reports, and helper state used by the Win32 frontend.

## Current Operational Contract

- Public UI: Firefox-first.
- Active downloader routes: `zeri`, `hentai2`, `hentaiaz`, `nyahentai`, `hitomi`.
- Browser-backed routes: `zeri`, `hentai2`, `hentaiaz`, `nyahentai`.
- HTTP-backed route: `hitomi`.
- Normal runtime root: `runtime\`.
- Portable runtime root: `portable-data\`.
- Per-task report: `tasks\task-<id>\report.json` under the active runtime root.
- Per-task thumbnail: `thumbnails\task-<id>\thumb.jpg` under the active runtime root.
- Download output: `<configured-output>\<manga-title>\`.
- Proxy setting: optional, persisted in frontend state, and passed to browser and backend HTTP clients.

## Frontend Behavior

- Adding a task starts it immediately.
- Failed, pending, paused, and queued tasks are retryable.
- When Firefox executable or Playwright driver settings are changed, all unfinished tasks have their stored browser fields refreshed.
- Completed tasks keep their original request metadata.
- The site filter is a dropdown beside the task-list title.
- The proxy menu persists a project-level proxy string used by task requests.

## Safety Review

No evidence was found for:

- Backdoor logic.
- Autostart or registry persistence.
- Remote control listener.
- Reverse shell or command-and-control behavior.
- Credential theft.
- Keylogging.
- Intentional exfiltration of local files.

Expected network behavior:

- Playwright opens user-supplied URLs.
- The downloader fetches image URLs collected by supported site parsers.
- Browser installer code can download Playwright-managed browser packages only when explicitly invoked from the app/tooling.
- Adblock rules are loaded from local rule files.

Expected persistence:

- Frontend settings.
- Task history and reports.
- Logs.
- Downloaded images.
- Thumbnails.
- Temporary browser profiles, which are removed after task completion when possible.
- Optional proxy server settings.

Residual risks and recommendations:

- Image downloads use HTTP requests; keep a maximum response-size limit on the roadmap.
- Logs include URLs and local paths; add optional log redaction if privacy requirements increase.
- Keep fresh task profiles as the default. Avoid workflows that copy real user Firefox profiles.
- Treat `dist\portable.exe` as a build artifact. Publish only trusted builds.
- `siteflow/assets.DownloadImagesContext` writes to the configured output root after sanitizing the title-derived folder name; callers should continue passing trusted local output roots.
- `netproxy.NormalizeServer` accepts proxy URLs with credentials because Go URL parsing allows them, but documentation should avoid credential examples.

## Documentation Status

- README reflects all active site flows, proxy policy, runtime layout, and portable layout.
- Interface flow reflects active Hitomi support, proxy propagation, shared assets, and browser-setting refresh.
- Flow rule documents exist for `zeri`, `hentai2`, `hentaiaz`, `nyahentai`, and `hitomi`.
- Browser profile flow documents the fresh temporary profile policy.
- Smoke tests use neutral placeholder URLs and avoid commands that bypass local execution policy.
- Security review is tracked in [`SECURITY_REVIEW.md`](SECURITY_REVIEW.md).
