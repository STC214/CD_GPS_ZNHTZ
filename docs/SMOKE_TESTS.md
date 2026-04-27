# Smoke Tests

This document records manual checks that are not covered by `go test ./...`, especially real Firefox launch, remote pages, and portable-build behavior.

## Before Running

Run unit tests first:

```powershell
go test ./...
```

Confirm local configuration:

- Firefox executable path can be set from the frontend menu or passed to `task-probe` with `-browser-path`.
- Playwright driver directory can be set from the frontend menu or passed to `task-probe` with `-driver-dir`.
- Runtime root defaults to `runtime\`, or can be overridden with `COMIC_DOWNLOADER_RUNTIME_ROOT`.

## Firefox Probe

Open a visible Firefox session:

```powershell
go run -tags playwright ./cmd/task-probe `
  -url "https://example.com" `
  -browser-type firefox `
  -headless=false `
  -keep-open=true
```

Expected:

- Firefox opens normally.
- The command prints the page title and temporary Playwright profile path.
- In Firefox, `about:support` shows `Application Basics -> Profile Directory` pointing to the printed temporary profile.
- Closing Firefox exits the command.

## Zeri Download Probe

Use a known working Zeri summary URL:

```powershell
go run -tags playwright ./cmd/task-probe `
  -url "https://www.zerobywzip.com/..." `
  -browser-type firefox `
  -headless=false `
  -download-root ".\runtime\smoke-output" `
  -output-dir ".\runtime\smoke-output"
```

Expected:

- The task parses the summary page and reader page.
- Images are downloaded under the configured output root and manga title.
- A task report is written under `runtime\tasks\`.
- A thumbnail is written under `runtime\thumbnails\`.

## Hentai2 Download Probe

Use a known working Hentai2 summary URL:

```powershell
go run -tags playwright ./cmd/task-probe `
  -url "https://www2.hentai2.net/kento-kun-wa-otona-no-omocha-ni-kyoumi-ga-atta-dake-de-mesu-ochi-suru-tsumori-wa-nakatta-youdesu/" `
  -browser-type firefox `
  -headless=false `
  -download-root ".\runtime\smoke-output" `
  -output-dir ".\runtime\smoke-output"
```

Expected:

- The task parses the summary title, page count, and `/read/...html` reader URL.
- Reader images are collected from the `.read1.text-center` area or sequentially expanded from the first image when applicable.
- Images are downloaded through `siteflow/assets`.
- A thumbnail is generated through `siteflow/assets`.
- The result contains `site=hentai2`.

## Win32 Frontend

Start the desktop app:

```powershell
go run -tags playwright ./cmd/win32-frontend
```

Expected:

- The window opens and restores saved settings.
- Adding a Zeri or Hentai2 URL creates and starts a task.
- Duplicate URLs prompt for confirmation.
- The site filter dropdown can switch between all tasks and site-specific tasks.
- Right-click task cards expose retry, details, open download directory, copy URL, delete, start, and pause.
- If a task failed before Firefox or driver was configured, setting Firefox/driver refreshes unfinished tasks so retry works without deleting and re-adding the task.

## Portable Build

Build and run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_portable.ps1
dist\portable.exe
```

Expected:

- `dist\portable.exe` starts the frontend.
- Persistent data is written under `dist\portable-data\`.
- Temporary `payload-*` directories are cleaned after exit.
- Logs are written under `dist\portable-data\logs\`.
