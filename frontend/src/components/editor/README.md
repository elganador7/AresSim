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

- Simplified: editor-only tasking constants and target-filtering rules now live in `utils/editorTasking.ts`. Live gameplay targeting no longer depends on frontend tasking heuristics.
- Remaining cleanup: `ScenarioEditor.tsx` is still a catch-all for scenario metadata, placement, selection, tasking, relationship editing, and serialization. Further additions should be split into smaller panels before the component becomes harder to review than to modify.
