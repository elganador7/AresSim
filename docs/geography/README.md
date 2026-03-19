# Geography Model

This document defines the replacement for the current simplified
`theaterCountries` / `CountriesAlongSegment(...)` approach.

The current logic is useful for a first theater prototype, but it conflates:

- land borders
- maritime jurisdiction
- sovereign airspace

That is not robust enough for Gulf, Red Sea, and Eastern Mediterranean
scenario design.

## Goal

Replace country-only route validation with a layered geographic model that can
answer:

- which country owns the land under a point
- whether a point is in territorial waters, EEZ, or high seas
- which country owns the sovereign airspace above a point
- whether a route crosses foreign airspace or international space

The first target is robust country / sea / airspace access validation.
Later features can build on the same layer for naval operations, civil
infrastructure, and diplomacy.

## Data Layers

The new lookup layer should use three polygon families.

### 1. Land Regions

Fields:

- `country_code`
- `name`
- `geometry`
- `source`

Purpose:

- land ownership lookup
- country placement checks
- land-based route validation

Recommended source:

- `geoBoundaries` ADM0 polygons

### 2. Maritime Regions

Fields:

- `owner_country_code`
- `zone_type`
- `geometry`
- `source`

`zone_type` values:

- `internal_waters`
- `territorial_sea`
- `contiguous_zone`
- `eez`
- `high_seas`

Purpose:

- distinguish territorial waters from international waters
- support naval movement and access rules
- support future maritime escalation / overflight logic

Recommended source:

- `Marine Regions`

### 3. Airspace Regions

Fields:

- `owner_country_code`
- `airspace_type`
- `min_alt_m`
- `max_alt_m`
- `geometry`
- `source`

`airspace_type` values for the first phase:

- `national`
- `international`

Possible later values:

- `fir`
- `adiz`
- `restricted`
- `military_ops`

Purpose:

- air transit validation
- strike-path validation
- country airspace access rules

Recommended source:

- phase 1: derived sovereign airspace from land + territorial sea
- later: simplified overlays derived from OpenAIP / AIXM-like sources

## Runtime Lookup Model

Add a dedicated package:

- `internal/geo`

Core types:

- `GeoContext`
- `GeoSegmentContext`
- `LookupIndex`

### `GeoContext`

Expected fields:

- `land_country`
- `sea_zone_owner`
- `sea_zone_type`
- `airspace_owner`
- `airspace_type`
- `is_international_waters`
- `is_international_airspace`

### `GeoSegmentContext`

Expected fields:

- `start`
- `end`
- `crossed_land_countries`
- `crossed_airspace_owners`
- `crossed_sea_zone_owners`
- `crossed_sea_zone_types`

## Runtime API

Replace current helpers with a small set of explicit geography APIs.

### Point Lookup

- `LookupPoint(lat, lon, altMsl) GeoContext`

Use cases:

- unit placement
- target classification by location
- defensive positioning checks

### Path Sampling

- `SamplePath(points []Point3D) []GeoSegmentContext`

Use cases:

- route validation
- strike-path validation
- future maritime lane checks

### Violation Search

- `FindFirstTransitViolation(...)`
- `FindFirstStrikeViolation(...)`
- `FindFirstPositioningViolation(...)`

These should return a structured result rather than a plain string so the UI
can explain the violation without duplicating logic.

## Rule Semantics

The geography layer is only the source of truth for where something is. Access
rules still come from the relationship matrix.

Examples:

- air transit checks use `airspace_owner`
- strike checks use `airspace_owner`
- future maritime access checks use `sea_zone_owner` and `zone_type`
- future naval high-seas behavior uses `is_international_waters`

## Phase 1 Scope

Phase 1 should stay intentionally narrow.

Implement:

- land ownership
- territorial waters
- high seas / international waters
- sovereign national airspace
- international airspace

Do not implement yet:

- FIRs
- ADIZ
- altitude-classified civil airspace
- dynamic NOTAMs
- military restricted zones

That is enough to replace the current theater-country approximation with
something materially better.

Current implementation status:

- airspace ownership: now derived from imported global country borders, with
  sovereign airspace still modeled as land + territorial sea only
- territorial waters vs international waters: now backed by the global 12nm
  Marine Regions GeoPackage import in `scripts/import_territorial_waters.py`
- EEZ / contiguous zone / internal waters: not yet implemented
- frontend border rendering: still visualization-only

## Preprocessing Pipeline

Do not depend on live third-party APIs at runtime.

Instead:

1. ingest official/open source boundary data
2. clip to the theater(s) of interest
3. simplify geometry for runtime use
4. normalize attributes to internal schema
5. emit local JSON / GeoJSON assets

Suggested outputs:

- `internal/geo/data/land_regions.json`
- `internal/geo/data/maritime_regions.json`
- `internal/geo/data/airspace_regions.json`

Suggested tooling:

- standalone scripts under `scripts/`
- deterministic preprocessing so generated assets can be reviewed in git

## Performance Strategy

Runtime lookups should not brute-force every polygon.

Use:

- bounding-box prefilter
- then point-in-polygon / segment sampling

If needed later:

- simple spatial grid index
- or R-tree style indexing

The initial theater scope is small enough that simplified regional datasets
should perform well with a modest index.

## Replacement Sequence

1. Add `internal/geo` types and loader.
2. Add preprocessed theater datasets.
3. Switch backend access validation:
   - transit
   - strike
   - defensive positioning
4. Switch frontend explainability to consume structured violation results.
5. Remove the old theater-country helpers once parity is confirmed.

## Frontend Impact

Current frontend helper still relevant here:

- `frontend/src/utils/countryRelationships.ts`

should eventually stop doing their own path ownership inference.

Preferred end state:

- backend computes authoritative path violations
- frontend renders the explanation and overlays
- frontend keeps only lightweight display helpers, not independent rule logic

## Open Design Notes

- Maritime access rules are not implemented yet, but this model should make
  them straightforward to add.
- Airspace in phase 1 is sovereignty-based, not aviation-control-based.
- This model is deliberately local-data-first to avoid runtime dependence on
  changing external services.

## Recommended Sources

- `geoBoundaries` for land borders
- `Marine Regions` for maritime zones
- `OpenAIP` only as an optional source/reference for later airspace overlays
- authoritative national / regional AIXM-like sources if the project later
  needs high-fidelity aviation constraints
