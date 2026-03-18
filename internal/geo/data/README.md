# Geo Data

This folder contains embedded backend geography data.

Current scope:

- `theater_borders.json`: canonical simplified theater border GeoJSON used for
  frontend border rendering and backend sovereign-airspace ownership lookup
- `theater_maritime.json`: canonical simplified territorial-water GeoJSON
  used for frontend maritime rendering and backend maritime ownership lookup

These datasets are intentionally coarse phase-1 assets. They should eventually
be replaced by preprocessed land / maritime / airspace data derived from the
sources described in [docs/geography/README.md](/Users/cameronspringer/AresSim/docs/geography/README.md).

Keep these files reviewable and deterministic. Do not hide important geometry
changes in generated binary formats.
