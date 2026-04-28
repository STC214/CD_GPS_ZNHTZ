# Interface Flow

This document describes the current Win32 UI and task contract.

## Input

- `url`
- `downloadRoot` / `outputDir`
- Firefox executable path
- Playwright driver directory
- Optional project proxy server

## Task Lifecycle

The task manager tracks these UI states:

- `pending`
- `queued`
- `routing`
- `preparing`
- `running`
- `paused`
- `waiting_verification`
- `verification_cleared`
- `completed`
- `failed`

Current frontend behavior:

- Clicking `添加任务` starts the task immediately.
- Duplicate tasks are detected by `URL + browser type` and prompt before adding.
- `Start all unfinished tasks` respects the configured concurrency value.
- `myreadingmanga.info` is blocked at add time and shows an unsupported-site message.
- The task-list site filter is a dropdown beside `任务列表`.
- The dropdown includes `显示全部`, `Zeri`, `Nyahentai`, `Hentai2`, `Hentaiaz`, and `Hitomi`.
- Active download flows currently exist for `zeri`, `hentai2`, `hentaiaz`, `nyahentai`, and `hitomi`.
- The proxy setting is optional. When configured, unfinished task requests are refreshed so retries use the new proxy.

## Runtime Files

Normal builds write under `runtime/`:

- `runtime/tasks/task-<id>/report.json`
- `runtime/logs/`
- `runtime/output/<manga-title>/`
- `runtime/thumbnails/task-<id>/thumb.jpg`
- `runtime/browser-profiles/`
- `runtime/frontend_state.json`
- `runtime/comic_downloader_state.json`

Portable builds write persistent data under `portable-data/`:

- `portable-data/tasks/task-<id>/report.json`
- `portable-data/logs/`
- `portable-data/output/<manga-title>/`
- `portable-data/thumbnails/task-<id>/thumb.jpg`
- `portable-data/browser-profiles/`
- `portable-data/frontend_state.json`
- `portable-data/comic_downloader_state.json`

The portable launcher creates temporary unpack directories inside `portable-data/payload-*` and removes them when it exits.

## UI Data Flow

- The task list is built from live task state plus report snapshots.
- Task details read `report.json` and append recent task-log content.
- The frontend persists window placement and current settings on exit.
- The frontend persists the current task list to the legacy history file on exit, so restart can restore current tasks.
- The browser menu includes Firefox executable and Playwright driver directory pickers.
- The settings menu includes a project proxy entry.
- After Firefox, driver, download directory, concurrency, progress refresh interval, or proxy settings are changed, unfinished task requests are refreshed with the new settings.
- Failed tasks that were created before Firefox or driver was configured can be retried directly after settings are saved.
- Task cards prefer the resolved manga title from the task result.
- The URL input uses a cue banner placeholder.

## Browser Data Flow

- The browser layer resolves a launch spec before launch.
- The launch spec includes browser type, browser path, install root, driver dir, temporary profile path, headless, keep-open, locale, timezone, viewport, user-agent, and optional proxy server.
- Firefox task runs use a fresh temporary Playwright profile per task.
- Browser middleware owns stealth injection, Firefox user prefs, adblock loading, proxy propagation, and launch defaults.
- The `task-probe` CLI remains the current browser smoke-test entry point.

## Site Flow

```text
frontend addPendingTask
-> ui.TodoList.RunImmediately
-> tasks.RunBrowserRequest
-> zeri / hentai2 / hentaiaz / nyahentai / hitomi URL dispatch
-> site parser resolves title, page count, reader URL, image URLs
-> siteflow/assets downloads images and creates thumbnail
-> BrowserRunResult updates the frontend task card
```

Active site contracts:

- `zeri`: documented in [`zeri_flow_rules.md`](zeri_flow_rules.md).
- `hentai2`: documented in [`hentai2_flow_rules.md`](hentai2_flow_rules.md).
- `hentaiaz`: documented in [`hentaiaz_flow_rules.md`](hentaiaz_flow_rules.md).
- `nyahentai`: documented in [`nyahentai_flow_rules.md`](nyahentai_flow_rules.md).
- `hitomi`: documented in [`hitomi_flow_rules.md`](hitomi_flow_rules.md); uses backend HTTP parsing and does not launch Playwright.

Shared asset contract:

- `siteflow/assets.DownloadImages` downloads images to `<output>/<manga-title>/`.
- `siteflow/assets.SelectThumbnailSource` chooses the best first-page candidate.
- `siteflow/assets.CreateJPGThumbnail` creates the task-card thumbnail.
- `siteflow/assets` uses the task proxy setting for backend HTTP image downloads.

## Progress

- Browser startup reports early progress around `0.02` and `0.08`.
- Site parsers report parse progress.
- Downloads are mapped into the final progress span based on the expected page count.
- The frontend coalesces rapid progress updates. The default refresh interval is `80ms`.
- The interval can be changed from the progress refresh menu and is persisted in frontend state.
