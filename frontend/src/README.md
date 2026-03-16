# Frontend Source

This folder contains the user-facing app code.

- `bridge/`: Wails event wiring and protobuf decoding.
- `components/`: Cesium map, editor UI, and HUD components.
- `store/`: Zustand state for editor and live sim.
- `utils/`: formatting, geography, and billboard helpers.
- `data/`: static UI data such as supported editor countries.
- `proto/`: generated TypeScript protobuf bindings.

When tracing a feature, start from `App.tsx`, then follow state through `store/` and event flow through `bridge/`.
