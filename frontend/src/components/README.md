# Components

This folder contains the main visual building blocks for the app.

- `CesiumGlobe.tsx`: live simulation map view.
- `UnitTypeIcon.tsx`: shared icon metadata and visual type helpers.
- `editor/`: scenario editing UI and editor map.
- `hud/`: live-sim overlays such as top bar, unit panel, and event log.

If a component starts mixing rendering, data shaping, and workflow logic, extract helpers into `utils/` or state transitions into `store/`.

## Review Notes

- `CesiumGlobe.tsx` is carrying too many responsibilities at once: Cesium bootstrapping, borders, route editing, waypoint dragging, visibility rules, strike-path legality, track links, munition FX, and imperative entity diffing. This file is now a major maintenance risk and should be split into focused sync modules or hooks before more map behaviors are added.
- Map interaction state is currently spread across `CesiumGlobe.tsx`, `UnitPanel.tsx`, `TopBar.tsx`, and the store. That distribution has already produced regressions around route mode and target-pick mode. A dedicated command-mode abstraction would reduce accidental coupling.
- The map re-creates many transient entities (`_route_seg_`, `_strike_seg_`, track links, waypoint markers) on every sync pass. It works, but it is an avoidable rendering cost that will get more expensive as scenario density increases.
