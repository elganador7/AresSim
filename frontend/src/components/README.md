# Components

This folder contains the main visual building blocks for the app.

- `CesiumGlobe.tsx`: live simulation map view.
- `UnitTypeIcon.tsx`: shared icon metadata and visual type helpers.
- `editor/`: scenario editing UI and editor map.
- `hud/`: live-sim overlays such as top bar, unit panel, and event log.

If a component starts mixing rendering, data shaping, and workflow logic, extract helpers into `utils/` or state transitions into `store/`.

## Review Notes

- Resolved in part: shared Cesium map policy and entity helpers now live in `components/cesium/helpers.ts`, so `CesiumGlobe.tsx` no longer carries all of the visibility, cursor, route-legality, and entity-construction logic inline.
- Remaining cleanup: `CesiumGlobe.tsx` still owns too many responsibilities at once: Cesium bootstrapping, borders, route editing, waypoint dragging, entity syncing, track links, and munition FX. It is materially better than before, but should still be split into focused sync modules or hooks before more map behaviors are added.
- Resolved in part: live map interaction now uses a single command-mode abstraction in the sim store, which removed the separate route-edit and target-pick flags that had been drifting across `CesiumGlobe.tsx`, `UnitPanel.tsx`, and `App.tsx`.
- The map re-creates many transient entities (`_route_seg_`, `_strike_seg_`, track links, waypoint markers) on every sync pass. It works, but it is an avoidable rendering cost that will get more expensive as scenario density increases.
