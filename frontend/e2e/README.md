# Frontend E2E Harness

This folder contains the browser-based frontend harness for AresSim.

What it does:
- starts the live Vite app
- launches a real Chromium browser with Playwright
- injects a browser-side Wails/runtime shim before the app boots
- drives the actual React UI like a user

This is not a `happy-dom` component test. It exercises the live frontend bundle in a real browser.

## Run

```bash
npm run --prefix frontend test:e2e
```

## Current coverage

- app shell boot
- oil network startup state in the top-bar menu
- scenario modal open/search surface
- switching from sim shell into the scenario editor

## Harness model

The browser harness installs:
- `window.go.main.App.*`
- `window.runtime.*`

These are test-side shims so the live frontend can run outside Wails while still using the real UI codepaths.

The helper lives in:
- [support/appHarness.ts](./support/appHarness.ts)

The first smoke suite lives in:
- [app.smoke.spec.ts](./app.smoke.spec.ts)
