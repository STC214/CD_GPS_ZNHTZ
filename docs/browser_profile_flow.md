# Browser Profile Flow

This repository currently uses a Firefox-only browser flow for the public UI and task layer.
Firefox tasks launch from a fresh temporary Playwright profile.

## Current Firefox launch flow

1. Resolve the configured Firefox executable path.
2. Create a brand-new temporary Playwright profile for the task.
3. Launch Firefox with `playwright-go` using the temp directory as `userDataDir`.
4. Inject `runtime/firefox_stealth.js` before any page script runs.
5. Open the target URL.
6. Wait until the browser window closes when `keep-open` is enabled.
7. Remove the temporary profile after the session ends.

## Current defaults

- Browser executable: user-configurable, with a system Firefox fallback in the code.
- Browser install root in this workspace: `runtime\playwright-browsers`
- Playwright driver directory in this workspace: `runtime\playwright-browsers\driver`
- Default Firefox User-Agent: `Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:149.0) Gecko/20100101 Firefox/149.0`
- Default locale: `en-US`
- Default timezone: `Asia/Shanghai`
- Default viewport: `1365x768`

## Direct browser entry

The browser middleware still accepts these request-time overrides:

- `profile-dir`
- `user-data-dir`
- `user-agent`
- `locale`
- `timezone-id`
- `viewport-width`
- `viewport-height`
- `firefox-user-prefs-json`
- `headless`
- `keep-open`

## Task-scoped profile directories

When a Firefox task needs its own profile, the runtime creates a fresh temp root under:

- `runtime/browser-profiles/tasks/firefox-fresh-*`

That temp root is disposable and is removed after the task ends.

## Why this matters

The goal is to keep the browser run reproducible while still starting from a temporary profile:

- Firefox task runs do not modify a shared mother profile in place.
- Each task gets its own temp profile.
- Temporary directories are cleaned at the end of the run.
- The browser middleware stays responsible for stealth injection and launch defaults.
- The portable build persists settings and history beside the executable in `portable-data/`.
- When the frontend saves a new Firefox executable path or Playwright driver directory, unfinished task requests are refreshed so retries use the new configuration.

## Useful browser self-check pages

These built-in pages are the fastest way to verify which profile a browser is actually using:

- `about:support`
  - Best for Firefox.
  - Check `Application Basics -> Profile Directory` to confirm the exact profile directory in use.
- `about:profiles`
  - Best for Firefox when you want to inspect all available profiles.
  - It shows the active profile and its `Root Directory` and `Local Directory`.

Use these pages when you need to confirm that a temporary profile is really the one being consumed by the browser.
