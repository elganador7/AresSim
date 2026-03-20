# Stores

This folder contains Zustand state for the editor and running simulation.

- `editorStore.ts`: scenario-authoring state, drafts, and editor country/team defaults.
- `simStore.ts`: live sim state, contacts, munitions, and map effects.

When debugging a UI issue, verify whether the bug is in store shape, bridge ingestion, or rendering before changing components.
