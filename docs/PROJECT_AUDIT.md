# Project Audit

Audit date: 2026-04-27

## Scope

This audit summarizes the current repository structure, supported site flows, runtime data layout, frontend/task contract, and the latest safety review. It is a static/local review and does not require live browser downloads.

## Verification

Command:

```powershell
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
- `runtime`: runtime paths, browser profile handling, frontend state, logging, and browser install helpers.
- `tasks`: task-level browser request normalization, site dispatch, progress mapping, downloading, thumbnails, and result reporting.
- `siteflow/assets`: shared image downloading, thumbnail source selection, and JPG thumbnail creation.
- `siteflow/zeri`: Zeri URL detection, summary parsing, reader parsing, pagination walking, and image URL collection.
- `siteflow/hentai2`: Hentai2 URL detection, summary parsing, reader parsing, lazy image collection, and sequential image expansion fallback.
- `siteflow/hentaiaz`: Hentaiaz URL detection, summary parsing, reader parsing, lazy image collection, and sequential image expansion fallback.
- `siteflow/nyahentai`: Nyahentai direct reader URL detection, reader-title parsing, lazy image collection, and reader-ID image filtering.
- `ui`: task list, legacy history, task reports, and helper state used by the Win32 frontend.

## Current Operational Contract

- Public UI: Firefox-first.
- Active downloader routes: `zeri`, `hentai2`, `hentaiaz`, `nyahentai`.
- UI filter placeholders: `hitomi`.
- Normal runtime root: `runtime\`.
- Portable runtime root: `portable-data\`.
- Per-task report: `tasks\task-<id>\report.json` under the active runtime root.
- Per-task thumbnail: `thumbnails\task-<id>\thumb.jpg` under the active runtime root.
- Download output: `<configured-output>\<manga-title>\`.

## Frontend Behavior

- Adding a task starts it immediately.
- Failed, pending, paused, and queued tasks are retryable.
- When Firefox executable or Playwright driver settings are changed, all unfinished tasks have their stored browser fields refreshed.
- Completed tasks keep their original request metadata.
- The site filter is a dropdown beside the task-list title.

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
- Browser installer code can download Playwright-managed browser packages when explicitly invoked.
- Adblock rules are loaded from local rule files.

Expected persistence:

- Frontend settings.
- Task history and reports.
- Logs.
- Downloaded images.
- Thumbnails.
- Temporary browser profiles, which are removed after task completion when possible.

Residual risks and recommendations:

- Image downloads use HTTP requests; keep timeout and maximum response-size limits on the roadmap.
- Logs include URLs and local paths; add optional log redaction if privacy requirements increase.
- Keep fresh task profiles as the default. Avoid copying real user Firefox profiles unless explicitly needed.
- Treat `dist\portable.exe` as a build artifact. Publish only trusted builds.

## Documentation Status

- README updated for zeri + hentai2.
- Interface flow updated for shared assets and browser-setting refresh.
- Zeri flow rules retained.
- Hentai2 flow rules added.
- Smoke tests rewritten with current commands.
