# Improvements Backlog

Current baseline: `go test ./...` and `npm test --prefix frontend` both pass. The items below are cleanup targets, ordered by payoff.

## Current Review Findings

Status:
- Resolved in current branch: 1, 2, 3, 4, 5, 6
- Still open: 7

### 1. Maritime placement validation still uses airspace ownership only
- Files: `app_sim.go:1165-1199`
- Severity: high
- Why: `PreviewDraftPlacement()` looks up only `geo.LookupPoint(...).AirspaceOwner` to determine the host country. That is correct for aircraft and land units, but it is wrong for `sea` and `subsurface` units now that the relationship model includes maritime transit/strike. A ship or submarine can still be placed in another state's territorial waters without consulting `SeaZoneOwner` or the maritime relationship flags.
- Cleanup:
  - Split placement validation by domain, the same way transit/strike validation already does
  - Use `SeaZoneOwner` + maritime rules for `sea` / `subsurface`
  - Add regression coverage for hostile, neutral, and defensive-only maritime placement cases
- Status: resolved

### 2. Relationship fallback logic is duplicated in the frontend after rule enforcement moved to the backend
- Files: `frontend/src/utils/countryRelationships.ts:1-154`, `frontend/src/components/hud/RelationshipPanel.tsx:38-54`, `frontend/src/components/editor/ScenarioEditor.tsx:827-883`, `internal/sim/relationships.go:136-173`
- Severity: medium-high
- Why: backend previews and live enforcement are now authoritative, but the frontend still keeps its own fallback policy in `getRelationshipRule()`. That means one future rule change has to be updated in two places, and the editor/live panels can display a different "default" relationship than the sim actually enforces. The old path-explanation helpers are also still present even though that responsibility moved to backend preview methods.
- Cleanup:
  - Make backend-produced relationship state the single source of truth for display defaults too
  - Remove dead frontend path-block explanation code once no callers remain
  - Keep the frontend helper limited to formatting, not policy
- Status: resolved for relationship-matrix rendering; frontend helper remains only for shared formatting/utilities

### 3. Country relationship UIs only know about countries that currently have units
- Files: `frontend/src/components/hud/RelationshipPanel.tsx:25-35`, `frontend/src/components/editor/ScenarioEditor.tsx:815-887`
- Severity: medium
- Why: both panels derive the country list from `units` only. Any explicit relationship rows for countries with no currently placed units disappear from the UI and cannot be reviewed or edited. That is a design mismatch for a diplomacy/access system, where permissions often need to exist before units are placed or before reinforcements arrive.
- Cleanup:
  - Build the country list from the union of:
    - countries present in units
    - countries already present in `relationships`
    - possibly scenario-supported countries later
  - Preserve explicit relationship rows even when there are no current units for one side
- Status: resolved

### 4. `app_sim.go` has become the next oversized hotspot
- Files: `app_sim.go:1-1199`
- Severity: medium
- Why: the file now mixes live sim control, relationship mutation, geometry preview APIs, JSON draft decoding, and transit/strike policy. The code is still understandable, but the geography/relationship feature set is large enough that `app_sim.go` is becoming the same kind of maintenance bottleneck the old `app.go` was.
- Cleanup:
  - Extract draft preview request parsing and preview handlers into an `app_sim_preview.go`
  - Extract relationship mutation and lookup helpers into an `app_relationships.go`
  - Keep `app_sim.go` focused on active-sim control and unit commands
- Status: partially resolved via `app_sim_preview.go`; further extraction is still optional

### 5. The relationship panel performs a scenario save and full resync on every single checkbox flip
- Files: `frontend/src/components/hud/RelationshipPanel.tsx:56-76`, `app_sim.go:674-718`
- Severity: medium
- Why: each toggle calls `SetCountryRelationship(...)`, persists the entire scenario, and then calls `RequestSync()`. Editing one country pair can therefore trigger six full save/resync cycles in a row. It works, but it is noisy, inefficient, and likely to feel laggy as the scenario grows.
- Cleanup:
  - Add local staging per relationship row and a single `Apply` action
  - Or add a batch update bridge method that updates all flags in one call without repeated full-state snapshots
- Status: resolved via staged row editing with explicit apply/reset

### 6. Generated and packaged artifacts are mixed into the working review set
- Files: `build/bin/AresSim.app/Contents/MacOS/AresSim`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/models.ts`, `frontend/src/proto/engine/v1/scenario_pb.ts`, `internal/gen/engine/v1/scenario.pb.go`
- Severity: medium
- Why: generated proto/Wails outputs are expected, but the packaged app binary under `build/bin/` is also tracked and modified. That makes review noisier and increases the chance of shipping a local build artifact in a source PR.
- Cleanup:
  - Exclude packaged build outputs from the review/PR unless intentionally releasing binaries
  - Keep generated code changes grouped with the source proto change that required them
- Status: packaged `build/bin/` output is now ignored and removed from source tracking

### 7. Geography data is now cleaner, but still duplicated across backend and frontend
- Files: `internal/geo/data/theater_borders.json`, `internal/geo/data/theater_maritime.json`, `frontend/src/data/theaterBorders.ts`, `frontend/src/data/theaterMaritime.ts`
- Severity: low-medium
- Why: moving the polygons to JSON was the right step, but the frontend and backend now each carry separate geometry copies. That improves code structure but still leaves data-drift risk, especially once polygon quality improves.
- Cleanup:
  - Choose one canonical geometry source and generate frontend render assets from it
  - Keep visualization-specific simplification as a build step, not as hand-maintained duplicate files
- Status: resolved for the current theater datasets; backend and frontend now read the same shared geo data files

## 1. Split `app.go` by responsibility
- Files: `app.go:93-228`, `app.go:421-1011`
- Why: `app.go` is over 1000 lines and mixes startup, DB seeding, bridge methods, DTO conversion, scenario loading, and event emission. That makes small changes risky and hard to test in isolation.
- Cleanup:
  - Move startup/shutdown and DB wiring into `app_lifecycle.go`
  - Move scenario control (`loadScenario`, `PauseSim`, `MoveUnit`, checkpoints) into `app_sim.go`
  - Move bridge serialization helpers (`scenarioRecord`, `decodeScenarioB64`, record-ID helpers) into `app_bridge_helpers.go`

## 2. Remove duplicated record/proto mapping logic in the backend
- Files: `app.go:159-228`, `app.go:668-847`, `app.go:903-989`
- Why: unit definition and weapon definition records are built and decoded inline in several places. Repeating `map[string]any` conversion logic increases drift risk every time the schema changes.
- Cleanup:
  - Introduce typed mapper helpers for `UnitDefinition` and `WeaponDefinition`
  - Centralize Surreal row decoding instead of open-coded `toFloat64`, `toString`, and `extractRecordID` use in multiple loops

## 3. Stop hard-coding development DB behavior in production code
- File: `internal/db/manager.go:39-43`
- Why: `UseInMemoryDB = true` makes persistence behavior depend on a compile-time constant. That is convenient for development, but it hides persistence bugs and makes release behavior less explicit.
- Cleanup:
  - Move storage mode to config or environment
  - Default to persisted `surrealkv` locally, with an explicit opt-in for in-memory mode

## 4. Extract frontend HUD pieces from `App.tsx`
- File: `frontend/src/App.tsx:23-438`
- Why: `App.tsx` owns route switching, top bar, event log, unit panel, map math helpers, and UI formatting. The component is readable now, but it is already a maintenance bottleneck.
- Cleanup:
  - Move `TopBar`, `EventLog`, and `UnitPanel` into `frontend/src/components/hud/`
  - Move `haversineM`, distance/ETA formatters, and side-color helpers into `frontend/src/utils/`

## 5. Reduce repeated full-collection scans in `CesiumGlobe`
- File: `frontend/src/components/CesiumGlobe.tsx:404-571`
- Why: stale-entity cleanup and munition cleanup repeatedly scan `viewer.entities.values`, build arrays, and filter them. This is acceptable at small scale but will get expensive as unit and munition counts rise.
- Cleanup:
  - Track known unit and munition entity IDs in local sets/maps
  - Replace repeated `Array.from(...).filter(...).forEach(...)` sweeps with incremental add/remove bookkeeping
  - Revisit whether `ListUnitDefinitions()` on mount should be cached centrally instead of reloaded per globe mount (`frontend/src/components/CesiumGlobe.tsx:503-517`)

## 6. Consolidate bridge event decoding
- File: `frontend/src/bridge/bridge.ts:37-284`
- Why: every Wails event repeats the same `fromBinary(..., b64ToBytes(b64))` error-handling pattern. That adds noise and makes new event types harder to add consistently.
- Cleanup:
  - Add a typed `decodeEvent(schema, b64, label)` helper
  - Add a small `registerEvent(name, schema, handler)` wrapper
  - Reuse one base64 utility across the frontend instead of duplicating byte conversion helpers here and in `ScenarioEditor.tsx:42-52`

## 7. Reduce unnecessary `Map` recreation churn in the Zustand store
- File: `frontend/src/store/simStore.ts:179-260`
- Why: store updates create fresh `Map` objects for nearly every mutation. That keeps subscriptions simple, but it also forces broad identity changes and full Cesium sync paths even for small updates.
- Cleanup:
  - Keep immutable updates, but group related writes into fewer store actions
  - Consider specialized actions for weapon updates and detection updates that preserve unchanged branches
  - Benchmark before changing semantics, because this is partly a deliberate tradeoff

## 8. Break up oversized editor components
- Files: `frontend/src/components/editor/ScenarioEditor.tsx` (752 lines), `frontend/src/components/editor/UnitDefinitionManager.tsx` (420 lines)
- Why: the editor files combine persistence, serialization, form state, and rendering. They are large enough that regressions will become hard to localize.
- Cleanup:
  - Split scenario metadata, placed-unit list, and save/load controls into separate components
  - Move draft-to-proto serialization out of the component and test it separately

## 9. Normalize logging and error handling across the frontend
- Files: `frontend/src/App.tsx`, `frontend/src/bridge/bridge.ts`, `frontend/src/components/editor/*.tsx`, `frontend/src/components/CesiumGlobe.tsx`
- Why: there is a mix of `console.log`, `console.warn`, and raw `console.error` calls. Errors are visible in development, but the calling code is inconsistent and difficult to filter.
- Cleanup:
  - Introduce a small frontend logger wrapper
  - Standardize user-visible failures vs debug-only diagnostics

## 10. Add focused tests around conversion boundaries
- Files: `app.go`, `frontend/src/bridge/bridge.ts`, `frontend/src/components/editor/ScenarioEditor.tsx`
- Why: most existing tests cover sim logic and store behavior. Serialization and bridge conversion code still relies heavily on manual inspection.
- Cleanup:
  - Add backend tests for record mappers and scenario encode/decode helpers
  - Add frontend tests for bridge event decoding and scenario draft serialization

## Suggested order
1. Split `app.go`
2. Consolidate backend mappers
3. Extract frontend HUD and editor serialization helpers
4. Consolidate bridge decoding
5. Optimize Cesium entity bookkeeping
