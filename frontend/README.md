# Frontend

This is the React + Vite + Cesium client used by Wails.

- `src/`: app source.
- `wailsjs/`: generated bridge bindings from the Go app. Do not edit by hand.
- `dist/`: production build output.

Most feature work lands in `src/`. Treat `wailsjs/` as generated and regenerate or rebuild rather than patching it manually.
