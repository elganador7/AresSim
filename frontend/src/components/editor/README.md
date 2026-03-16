# Editor Components

This folder contains the scenario editor workflow.

- `ScenarioEditor.tsx`: overall editor shell and scenario serialization.
- `EditorGlobe.tsx`: placement map.
- `UnitPalette.tsx`: searchable country-filtered library browser.
- `DropConfirmDialog.tsx`: placement confirmation and loadout selection.
- `UnitDefinitionManager.tsx`: CRUD UI for unit definitions.
- `editor.css`: editor-specific styling.

This area is the main authoring surface for scenarios. Prefer keeping editor-specific behavior here rather than leaking it into the live-sim HUD.
