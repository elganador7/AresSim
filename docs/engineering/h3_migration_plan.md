# H3 Migration Plan

## Goal
Transition the repo from raw latitude/longitude as the primary geographic storage primitive to Uber H3 cells, using:

- `resolution 12` as the default precision target
- `resolution 11` as the approved fallback for broader overlays or lower-signal data

The intent is not to remove latitude/longitude entirely. Cesium, external datasets, and some simulation math still need geographic coordinates. The migration should make H3 the canonical storage/indexing key while treating lat/lon as derived or compatibility data.

## Default Resolution Policy

- Default simulation/storage precision: `H3 r12`
- Coarser visualization/aggregation precision: `H3 r11`
- Rule:
  - store the finest supported canonical cell for authored/runtime objects at `r12`
  - derive `r11` parents for aggregation, culling, caches, and low-detail views

## Constraints

- Backend official library: `github.com/uber/h3-go/v4`
- Frontend official library: `h3-js`
- `h3-go` uses CGO-backed bindings, so H3 should be centralized behind a small internal compatibility layer
- Cesium still requires lon/lat for rendering and picking
- Existing protobuf schemas and Wails bridge types are lat/lon-centric today

## Migration Principles

1. Do not do a big-bang rewrite.
2. Introduce H3 alongside lat/lon first.
3. Make H3 canonical in domain/storage layers before forcing every UI path to change.
4. Keep lat/lon as derived compatibility fields until bridge/proto/UI migration is complete.
5. Add focused tests per subsystem before replacing raw coordinate assumptions.

## Target Model

### Canonical Geo Identity

Every location-bearing runtime/domain object should eventually support:

- `h3CellR12`
- optional `h3CellR11` parent for aggregation
- optional derived centroid lat/lon
- optional exact geometry/path for rendering or routing

### Lat/Lon After Migration

Lat/lon remains useful for:

- Cesium rendering
- route geometry
- polygon boundaries
- imported datasets
- exact movement physics where needed

But it should stop being the primary identity and lookup key for point features.

## Phased PR Sequence

### PR A: Shared H3 Geometry Foundation

- add backend H3 dependency and helper layer
- add `geo.H3Cell` abstraction and resolution policy
- extend `geo.Point` with optional canonical H3 cell
- add conversion helpers:
  - point -> h3
  - h3 -> centroid point
  - r12 -> r11 parent
- add tests around normalization, parsing, and round-trips

### PR B: Protocol And Bridge Compatibility

- add optional H3 cell fields to shared proto/Wails-facing models
- keep existing lat/lon fields for compatibility
- update bridge normalization to propagate both
- add bridge tests for H3-backed payloads

### PR C: Scenario And Authoring Model Migration

- update scenario helpers and authored objects to set canonical H3 cells
- keep lat/lon constructor inputs initially, but derive/store H3 immediately
- add helper constructors for H3-native authored points later

### PR D: Oil Network Migration

- make oil nodes canonical on H3 cells
- keep route geometries as lat/lon paths
- use r11 parents for aggregation, global culling, and cache bucketing
- update render-cache schema to include H3 cell metadata

### PR E: Simulation Runtime Migration

- units, munitions, explosions, and waypoints get canonical H3 fields
- retain exact lat/lon for movement and render compatibility
- update proximity/lookups/culling opportunities to use H3 where beneficial
- do not replace great-circle movement math with hex stepping unless there is a specific reason

### PR F: Frontend Type System Migration

- add `h3Cell` to TypeScript location-bearing types
- add frontend H3 helpers via `h3-js`
- use H3 parents for view bucketing, clustering, and density control
- preserve Cesium lon/lat rendering path

### PR G: Editor And UI Migration

- editor drafts carry canonical H3 cell
- globe picking converts clicked lat/lon -> H3
- panels can show both precise coordinates and cell IDs when useful

### PR H: Cleanup

- remove remaining lat/lon-as-identity assumptions
- keep lat/lon only where exact geometry is required
- document final geo model and supported resolutions

## Subsystem-Specific Plan

### 1. `internal/geo`

First subsystem to change.

- introduce the H3 abstraction layer here
- keep polygon lookup code on exact lon/lat for now
- add helpers for:
  - `Point.EnsureH3`
  - `Point.CanonicalCell`
  - `ParentCell`
  - `CentroidPoint`

### 2. Protobuf / Wails

Current schemas are lat/lon-only.

Migration path:

- add `string h3_cell = N` fields to `Position` and `Waypoint`
- regenerate bindings
- keep `lat`/`lon` for compatibility during transition
- define backend rule:
  - if H3 is absent, derive it from lat/lon
  - if lat/lon are absent later, derive centroid from H3 for compatibility

### 3. Simulation

Do not rewrite all motion into hex steps.

Use H3 for:

- spatial indexing
- coarse neighborhood/proximity buckets
- aggregation/culling
- future contact acceleration

Keep exact lat/lon for:

- movement interpolation
- bearings
- great-circle distance
- render output

### 4. Oil Network

Best near-term beneficiary.

- nodes become H3-native
- routes remain polylines
- use r11 parents for map-level aggregation and caching
- precompute:
  - node `h3_r12`
  - node `h3_r11`
  - route point cells only if later needed for spatial joins

### 5. Frontend / Cesium

Cesium remains lon/lat-based.

Frontend H3 should be used for:

- bucketing
- clustering
- selection/context aggregation
- large-overlay visibility decisions

Not for:

- direct rendering coordinates

### 6. Editor

- picked terrain point => lat/lon => canonical H3
- drafts store both during transition
- serialized drafts/scenarios should gradually become H3-aware

## Test Strategy

### Backend

- `internal/geo` unit tests for H3 conversion and resolution normalization
- bridge tests for H3 propagation
- scenario tests ensuring authored points get canonical cells
- oilnet tests ensuring nodes carry H3 and aggregation parents

### Frontend

- Type normalization tests in bridge/store
- oil-layer bucketing tests on H3 parents once introduced
- editor tests for click -> H3 conversion

## Efficiency Opportunities Unlocked By H3

- cheaper global oil-layer bucketing using `r11`
- future unit/contact spatial indexing by cell neighborhood
- faster dataset reconciliation by cell instead of raw coordinate tolerance
- more stable cache keys for oil and scenario overlays

## Immediate Execution Plan

1. Add shared backend H3 helpers and tests.
2. Add optional H3 cell to `geo.Point`.
3. Start propagating H3 into oilnet nodes and route-independent point objects.
4. Add protocol/bridge compatibility fields.
5. Add frontend H3 support only after backend data is present.

## References

- Uber H3 official Go bindings: https://github.com/uber/h3-go
- Uber H3 official JavaScript bindings: https://github.com/uber/h3-js
