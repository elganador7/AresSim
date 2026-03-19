# Geo Data

This folder contains embedded backend geography data.

Current scope:

- `world_borders.json`: canonical global country-border GeoJSON derived from
  `ne_10m_admin_0_countries/ne_10m_admin_0_countries.shp`
- `theater_borders.json`: theater subset generated from the same border source
  for frontend border rendering
- `world_12nm.json`: canonical global territorial-water GeoJSON derived from
  `World_12NM_v4_20231025_gpkg/eez_12nm_v4.gpkg`
- `theater_maritime.json`: theater subset generated from the same 12nm source
  for frontend overlay rendering

Generation:

- `python3 scripts/import_country_borders.py`
- `python3 scripts/import_territorial_waters.py`

The raw source datasets are local inputs and are intentionally not tracked in
git:

- `ne_10m_admin_0_countries/`
- `World_12NM_v4_20231025_gpkg/`

This import step makes the GeoPackage the source of truth for maritime data.
Do not hand-edit the generated JSON files unless you are also updating the
import pipeline.

Keep these files reviewable and deterministic. Do not hide important geometry
changes in generated binary formats.
