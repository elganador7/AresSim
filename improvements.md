# Improvements Backlog

Current baseline: `go test ./...` and `npm test --prefix frontend` both pass. The items below are cleanup targets, ordered by payoff.

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
