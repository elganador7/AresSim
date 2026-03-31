# Oil Network Layer

This folder tracks the architecture for the global civilian oil-trade layer.

## Goal

Represent the global oil system as a separate infrastructure graph that can be
rendered on the map, queried by component, and shocked by outages without
reusing military objects.

At runtime the map layer should support:

- global view of extraction, terminals, pipelines, refineries, storage, and chokepoints
- clickable components with daily throughput
- refinery output slates for derivatives such as gasoline, diesel, and jet fuel
- recomputation of supply shocks when a component goes degraded or offline

## Domain Model

The new domain lives in `internal/oilnet/`.

There is now also a dedicated maritime petroleum submodel in:

- `internal/oilnet/maritime/`

- `Node`
  - `extraction_site`
  - `gathering_hub`
  - `marine_terminal`
  - `export_terminal`
  - `import_terminal`
  - `refinery`
  - `storage_hub`
  - `demand_center`
  - `chokepoint`
  - `canal`
- `Edge`
  - `pipeline`
  - `shipping_lane`
  - `seaborne_corridor`
  - `internal_transfer`
- `Commodity`
  - `crude`
  - `gasoline`
  - `diesel`
  - `jet_kerosene`
  - `fuel_oil`
  - `lpg`
  - `naphtha`

## Accuracy Strategy

Use a layered source model:

1. Topology sources
- Global Energy Monitor GOGET
- Global Energy Monitor GOIT

2. Throughput and balancing sources
- JODI Oil
- UN Comtrade
- EIA for U.S. detail

3. Premium precision sources if available later
- Kpler
- Vortexa
- S&P / Platts
- Wood Mackenzie

Every node and edge should keep provenance metadata:

- source name
- URL
- last updated
- confidence score

## Implementation Phases

### Phase 1

- standalone `internal/oilnet` package
- embedded baseline global graph
- backend bridge method to fetch the graph
- map-layer-ready node and edge payloads

### Phase 2

- graph balancing and flow recomputation engine
- outage state changes and shock recomputation
- regional product shortfall outputs

This phase is now started:

- `SimulateShock(...)` recomputes the baseline graph after node or edge outages
- chokepoints can collapse downstream shipping flows
- refinery derivative outputs scale down with intake loss
- country and commodity summaries report lost flow and unmet demand

Current limitation:

- the solver is a deterministic baseline allocator, not yet a full min-cost rerouting optimizer
- it is good enough for first-order outage modeling and UI drilldown
- later phases should add rerouting, substitution, storage drawdown, and multi-day recovery

### Phase 3

- frontend layer and click panels
- commodity filters
- shock mode overlays

### Phase 4

- higher-fidelity ingest
- refinery-specific yields
- vessel and terminal flow precision

The ingest foundation is now started in:

- `internal/oilnet/ingest/`

Current adapters:

- GEM topology CSV normalization
- JODI country-balance CSV normalization
- UN Comtrade corridor-flow CSV normalization

These adapters are intentionally local-file based first. The goal is to define
stable normalized records and test them before adding fetch jobs or scheduled
updates.

There is now also an end-to-end sample path:

- `LoadSampleGraph()` builds a graph from embedded sample GEM/JODI/Comtrade CSVs

This is the bridge from hand-authored baseline data to generated topology. The
next step is to replace larger regions of `global_baseline.json` with generated
subgraphs from real exports.

Current generated regional overlays:

- Gulf crude production and export corridor
- Northwest Europe refining/import corridor
- Northeast Asia demand/import corridor

Pipeline infrastructure is now sourced from the local GeoJSON dataset directly:

- `data/GEM-GOIT-Oil-NGL-Pipelines-2025-03.geojson`

The parser lives in:

- `internal/oilnet/ingest/pipelines_geojson.go`

And the interpretation script lives in:

- `cmd/ingest-oil-pipelines/main.go`

This avoids hand-converting the pipeline geometry. The oil graph now consumes
pipeline routes from the source GeoJSON and keeps their real line geometry for
later map overlay work.

Extraction-site locations are now sourced from the local GOGET workbook:

- `data/Global-Oil-and-Gas-Extraction-Tracker-March-2026.xlsx`

The parser lives in:

- `internal/oilnet/ingest/extraction_xlsx.go`

And the interpretation script lives in:

- `cmd/ingest-oil-extraction/main.go`

The parser reads the `Field-level main data` sheet directly and extracts field
locations, fuel type, status, operator, and source URLs into extraction-site
overlay nodes.

## Maritime Topology

The first maritime petroleum slice now exists and is wired into the live oil
runtime graph.

Current files:

- `internal/oilnet/maritime/model.go`
- `internal/oilnet/maritime/ingest/topology.go`
- `internal/oilnet/maritime/ingest/topology_file.go`
- `internal/oilnet/maritime/ingest/data/topology_seed.json`

What the current slice includes:

- marine terminals
- canals
- seaborne corridors
- tanker-class metadata
- corridor chokepoint dependencies

Runtime loading now prefers:

1. `data/oil-maritime-topology.json`
2. embedded seed topology fallback

This keeps the app runnable today while giving us a clean ingest seam for real
terminal and chokepoint datasets.

Validation and conversion command:

- `cmd/ingest-oil-maritime-topology`

Example:

- `go run ./cmd/ingest-oil-maritime-topology --input data/oil-maritime-topology.json`

## Source Validation

The real-data parsers now fail explicitly when the source exports drift away
from the shapes the app depends on.

Extraction workbook validation:

- required sheets:
  - `Field-level main data`
  - `Field-level production data`
  - `Field-level reserves data`
  - `Project-level main data`
  - `Project-level production data`
- required columns on the main sheets:
  - field main:
    - `Unit ID`
    - `Unit Name`
    - `Fuel type`
    - `Country/Area`
    - `Status`
    - `Latitude`
    - `Longitude`
  - project main:
    - `Project ID`
    - `Project Name`
    - `Fuel type`
    - `Country/Area`
    - `Status`

Pipeline GeoJSON validation:

- root `type` must be `FeatureCollection`
- each parsed feature must provide:
  - `ProjectID`
  - `Fuel`
- geometry must still decode into at least one valid line route

These checks are meant to fail loudly when upstream exports change, instead of
silently producing bad graphs.

## Render Cache

The render cache is now a wrapped JSON file, not just a raw graph blob.

Current schema:

- `metadata`
  - `schemaVersion`
  - `builtAt`
  - `graphId`
  - `graphVersion`
- `diagnostics`
  - total node and edge counts
  - counts by node kind / edge kind
  - counts by node state / edge state
  - primary commodity counts
- `graph`
  - the renderable graph payload consumed by the app

Current schema version:

- `oil-renderable-cache/v1`

The app still accepts older raw graph JSON caches for backward compatibility,
but the cache builder now writes the wrapped format by default.

## Recommended Source Stack

- Global Energy Monitor GOGET  
  https://globalenergymonitor.org/projects/global-oil-gas-extraction-tracker/
- Global Energy Monitor GOIT  
  https://globalenergymonitor.org/projects/global-oil-infrastructure-tracker/
- JODI Oil  
  https://www.jodidata.org/oil/database/overview.aspx
- UN Comtrade API  
  https://comtradedeveloper.un.org/apis
- EIA Petroleum Data  
  https://www.eia.gov/petroleum/data.php
- EIA Refinery Capacity  
  https://www.eia.gov/petroleum/refinerycapacity

## Current Status

The repository now includes a starter global baseline graph in:

- `internal/oilnet/data/global_baseline.json`

It is intentionally a baseline, not the final authoritative dataset. The point
of this first slice is to fix the object model and API shape so the data can
grow without changing the rest of the app every week.
