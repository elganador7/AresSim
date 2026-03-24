# HUD Components

This folder contains the live simulation overlays layered above the map.

- `TopBar.tsx`: sim controls and status.
- `ViewSwitcher.tsx`: main view toggles.
- `EventLog.tsx`: event feed.
- `UnitPanel.tsx`: selected-unit details.

These components should stay compositional and presentation-focused. Keep the authoritative state in `store/`.

## Review Notes

- Simplified: live HUD targeting no longer depends on frontend tasking helpers. Engagement eligibility is backend-owned and the HUD now renders backend results.
- Resolved in part: live map interaction now runs through a single command-mode object in the sim store instead of separate route-edit and target-pick flags spread across HUD and map code.
- Remaining cleanup: `UnitPanel.tsx` still owns a lot of workflow orchestration and backend command plumbing. It is more stable now, but it should eventually lose more of that responsibility to smaller tasking components or store actions.
- Resolved in part: the country-relationship matrix now lives in `RelationshipPanel.tsx` instead of being embedded directly inside `TopBar.tsx`, so sim controls and diplomacy/access controls are no longer coupled in one component.
- Remaining cleanup: the relationship panel itself may still need its own route/state helpers if more diplomacy features are added, but the top-bar split is now in place.
