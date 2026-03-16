# Components

This folder contains the main visual building blocks for the app.

- `CesiumGlobe.tsx`: live simulation map view.
- `UnitTypeIcon.tsx`: shared icon metadata and visual type helpers.
- `editor/`: scenario editing UI and editor map.
- `hud/`: live-sim overlays such as top bar, unit panel, and event log.

If a component starts mixing rendering, data shaping, and workflow logic, extract helpers into `utils/` or state transitions into `store/`.
