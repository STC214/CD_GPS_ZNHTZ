# Historical Planning Note

This file is kept as a historical note for the earlier refactor plan.
The current authoritative docs are `README.md` and the files under `docs/`.

## What is true now

- The public UI is Firefox-first.
- Browser implementation is Firefox-only for the public UI and task layer.
- Firefox tasks use a fresh temporary profile per task and clean it on exit.
- Zeri is the active supported site flow.
- `myreadingmanga.info` is currently blocked in the frontend task-add flow.
- Portable builds persist settings and history beside the executable in `portable-data/`.

## Historical notes

The original plan focused on:

1. A Go + Playwright browser skeleton.
2. Firefox middleware, task isolation, and state persistence.
3. Stealth injection and profile copying rules.
4. Task routing and site-worker linkage.

The plan itself is still useful as a reminder of the architecture goals, but it should no longer be treated as the source of truth for current runtime behavior.

## Completed / Addressed

- [x] Fix app icon embedding so the exe and Win32 titlebar/taskbar use the embedded icon consistently.
- [x] Audit task-add request payloads and align frontend-to-backend URL-only submission for task creation.
- [x] Convert the URL input placeholder into background hint text instead of a visible text layer.
- [x] Fix task card rendering during window resize.
- [x] Add spacing between the URL input and the adjacent button.
- [x] Persist task state/history on exit so imported tasks survive restart without re-import.

