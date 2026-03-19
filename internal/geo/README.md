# Geo

This package is the backend geographic lookup layer.

Current scope:

- global land ownership and sovereign-airspace lookup derived from Natural
  Earth country borders
- global 12nm territorial-water lookup derived from the Marine Regions
  GeoPackage in `World_12NM_v4_20231025_gpkg/`
- path sampling for transit / strike validation

This package is the first step in replacing the older country-only helper logic
that previously lived in `internal/sim/theater_countries.go`.

Planned expansion:

- separate land ownership from airspace ownership
- add richer maritime zones beyond territorial waters
- return structured violation contexts for frontend explainability

Do not add scenario or diplomacy logic here. This package should only answer
geographic ownership / context questions.
