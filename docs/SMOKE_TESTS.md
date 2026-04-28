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
- Proxy is optional. Prefer setting it from the frontend when you need to verify proxy behavior.

## Firefox Probe

Open a visible Firefox session with a neutral URL:

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
- Closing Firefox exits the command.

## Supported Site Download Probe

Use a known working supported-site URL. Keep the URL local to the test environment and do not commit private, account-gated, or age-gated URLs into documentation.

```powershell
go run -tags playwright ./cmd/task-probe `
  -url "https://example.test/supported-gallery" `
  -browser-type firefox `
  -headless=false `
  -download-root ".\runtime\smoke-output" `
  -output-dir ".\runtime\smoke-output"
```

Expected:

- The task dispatches to the expected site flow.
- Browser-backed sites parse the summary/reader pages through Firefox.
- Hitomi parses gallery metadata through the HTTP resolver.
- Images are downloaded under the configured output root and manga title.
- A task report is written under `runtime\tasks\`.
- A thumbnail is written under `runtime\thumbnails\`.

## Proxy Probe

Configure a test proxy from the frontend, then run a supported-site task. For CLI-only checks, persist the proxy in frontend state first or set the process environment before launching the probe.

Expected:

- Browser navigation uses the configured proxy for browser-backed flows.
- Backend HTTP downloads use the configured proxy.
- Empty proxy settings fall back to normal system/environment behavior.

## Win32 Frontend

Start the desktop app:

```powershell
go run -tags playwright ./cmd/win32-frontend
```

Expected:

- The window opens and restores saved settings.
- Adding a supported-site URL creates and starts a task.
- Duplicate URLs prompt for confirmation.
- The site filter dropdown can switch between all tasks and site-specific tasks.
- Right-click task cards expose retry, details, open download directory, copy URL, delete, start, and pause.
- If a task failed before Firefox or driver was configured, setting Firefox/driver refreshes unfinished tasks so retry works without deleting and re-adding the task.

## Portable Build

Build and run:

```powershell
powershell -File .\scripts\build_portable.ps1
dist\portable.exe
```

Expected:

- `dist\portable.exe` starts the frontend.
- Persistent data is written under `dist\portable-data\`.
- Temporary `payload-*` directories are cleaned after exit.
- Logs are written under `dist\portable-data\logs\`.
