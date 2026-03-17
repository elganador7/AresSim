# HUD Components

This folder contains the live simulation overlays layered above the map.

- `TopBar.tsx`: sim controls and status.
- `ViewSwitcher.tsx`: main view toggles.
- `EventLog.tsx`: event feed.
- `UnitPanel.tsx`: selected-unit details.

These components should stay compositional and presentation-focused. Keep the authoritative state in `store/`.

## Review Notes

- `UnitPanel.tsx` now owns significant workflow logic: target filtering, order editing, route editing, warning generation, and backend command orchestration. That is beyond a presentation panel and is why regressions like hook-order issues and form-reset behavior have already appeared here.
- `TopBar.tsx` now mixes sim controls with a dense relationship-matrix editor. It works, but it is likely the wrong long-term home for that feature. Country relationships should probably move into their own panel before more flags and diplomacy controls are added.
