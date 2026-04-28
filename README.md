# Comic Downloader

A Windows comic downloader built with Go, Playwright Firefox, and a native Win32 frontend.

The application accepts comic page URLs, resolves site-specific image sources, downloads the target images, writes task reports, and creates task-card thumbnails.

## Current Status

- Public UI: Win32 desktop frontend.
- Browser runtime: Firefox through Playwright.
- Browser profile policy: fresh temporary profile per task, cleaned after the task leaves the running state.
- Active site flows:
  - `zeri`
  - `hentai2`
  - `hentaiaz`
  - `nyahentai`
  - `hitomi`
- Unsupported site blocked at add time:
  - `myreadingmanga.info`
- Shared asset pipeline:
  - `siteflow/assets` downloads all resolved image URLs.
  - The download stage uses adaptive workers up to 7 concurrent image downloads.
  - Thumbnails are generated as JPG files for the task board.
- Portable build:
  - Single file: `dist\portable.exe`
  - Persistent data: `dist\portable-data\`

## Quick Start

Run tests:

```powershell
$env:GOTMPDIR=(Join-Path (Get-Location) '.gotmp')
$env:GOCACHE=(Join-Path (Get-Location) '.gocache')
go test ./...
```

Run the Win32 frontend:

```powershell
go run -tags playwright ./cmd/win32-frontend
```

Run a visible task probe:

```powershell
go run -tags playwright ./cmd/task-probe `
  -url "https://example.com" `
  -headless=false `
  -keep-open=true
```

Build the portable single-file executable:

```powershell
powershell -File .\scripts\build_portable.ps1
```

Run the portable build:

```powershell
dist\portable.exe
```

## Runtime Layout

Normal development runs use `runtime\`:

- `runtime\tasks\task-<id>\report.json`
- `runtime\logs\`
- `runtime\output\`
- `runtime\thumbnails\task-<id>\thumb.jpg`
- `runtime\browser-profiles\`
- `runtime\frontend_state.json`
- `runtime\comic_downloader_state.json`

Portable runs use `dist\portable-data\`:

- `dist\portable-data\tasks\task-<id>\report.json`
- `dist\portable-data\logs\`
- `dist\portable-data\output\`
- `dist\portable-data\thumbnails\`
- `dist\portable-data\browser-profiles\`
- `dist\portable-data\frontend_state.json`
- `dist\portable-data\comic_downloader_state.json`

The portable launcher extracts a temporary payload into `dist\portable-data\payload-*` while it is running and removes that payload directory on exit.

## Browser And Driver Paths

The frontend can configure:

- Firefox executable path.
- Playwright browser root.
- Playwright driver directory.

If the browser and driver are both under one Playwright root such as:

```text
D:\Program\playwright-browsers
```

selecting that root lets the frontend auto-detect the Firefox executable and `driver` directory.

Changing Firefox or driver settings refreshes unfinished tasks so retrying a failed task uses the new runtime configuration.

## Environment Variables

- `COMIC_DOWNLOADER_WORKSPACE_ROOT`: override workspace root.
- `COMIC_DOWNLOADER_RUNTIME_ROOT`: override runtime root.
- `COMIC_DOWNLOADER_DOWNLOAD_DIR`: override default output directory.
- `COMIC_DOWNLOADER_FRONTEND_STATE_PATH`: override frontend settings path.
- `COMIC_DOWNLOADER_STATE_PATH`: override legacy task-history state path.
- `PLAYWRIGHT_BROWSERS_PATH`: override Playwright browser root.
- `PLAYWRIGHT_DRIVER_PATH`: override Playwright driver path.
- `COMIC_DOWNLOADER_PROXY`: optional HTTP/HTTPS/SOCKS proxy for backend HTTP downloads when the frontend setting is empty.

## Site Flow Summary

```text
frontend addPendingTask
-> ui.TodoList.RunImmediately
-> tasks.RunBrowserRequest
-> site-specific parser or HTTP resolver
-> siteflow/assets downloads target images
-> siteflow/assets creates task thumbnail
-> BrowserRunResult updates task history and frontend cards
```

Browser-backed site flows:

- `zeri`
- `hentai2`
- `hentaiaz`
- `nyahentai`

HTTP resolver site flow:

- `hitomi`

Hitomi does not need Playwright for parsing. It resolves `galleries/{id}.js`, applies the Hitomi `gg.js` hash rules, generates CDN image URLs, and then hands the image list to the shared download pipeline.

## Proxy Policy

Proxy settings are optional. When configured from the frontend, the same normalized proxy value is used by browser-backed flows and backend HTTP downloads. When the setting is empty, browser and HTTP clients use their normal default behavior.

Supported proxy schemes are:

- `http`
- `https`
- `socks4`
- `socks4a`
- `socks5`
- `socks5h`

## Reference Source

The local reference checkout is intentionally ignored by git:

```text
references\hitomi-downloader\
```

It is used only to compare Hitomi's parsing and image URL rules. Do not copy its repository files into this project wholesale.

## Documentation Index

- [Project audit](docs/PROJECT_AUDIT.md)
- [Security review](docs/SECURITY_REVIEW.md)
- [Interface flow](docs/INTERFACE_FLOW.md)
- [Browser profile flow](docs/browser_profile_flow.md)
- [Zeri flow rules](docs/zeri_flow_rules.md)
- [Hentai2 flow rules](docs/hentai2_flow_rules.md)
- [Hentaiaz flow rules](docs/hentaiaz_flow_rules.md)
- [Nyahentai flow rules](docs/nyahentai_flow_rules.md)
- [Hitomi flow rules](docs/hitomi_flow_rules.md)
- [Smoke tests](docs/SMOKE_TESTS.md)
