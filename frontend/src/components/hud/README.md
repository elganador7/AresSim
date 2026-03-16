# HUD Components

This folder contains the live simulation overlays layered above the map.

- `TopBar.tsx`: sim controls and status.
- `ViewSwitcher.tsx`: main view toggles.
- `EventLog.tsx`: event feed.
- `UnitPanel.tsx`: selected-unit details.

These components should stay compositional and presentation-focused. Keep the authoritative state in `store/`.
