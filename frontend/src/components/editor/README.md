# Editor Components

This folder contains the scenario editor workflow.

- `ScenarioEditor.tsx`: overall editor shell and scenario serialization.
- `EditorGlobe.tsx`: placement map.
- `UnitPalette.tsx`: searchable country-filtered library browser.
- `DropConfirmDialog.tsx`: placement confirmation and loadout selection.
- `UnitDefinitionManager.tsx`: CRUD UI for unit definitions.
- `editor.css`: editor-specific styling.

This area is the main authoring surface for scenarios. Prefer keeping editor-specific behavior here rather than leaking it into the live-sim HUD.

## Review Notes

- `ScenarioEditor.tsx` now duplicates a large amount of command/tasking logic that also exists in `hud/UnitPanel.tsx`: attack-task fields, route mode controls, target validation, and order semantics. This is a DRY problem and is already making behavior changes land twice.
- Editor-only concerns and live-scenario concerns are starting to drift. The next cleanup pass should extract shared tasking components and validation helpers so the editor and HUD cannot silently diverge on what an order means.
- The file has become the catch-all for scenario metadata, placement, selection, tasking, relationship editing, and serialization. Further additions should be split into smaller panels before the component becomes harder to review than to modify.
