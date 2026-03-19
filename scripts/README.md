# Scripts

This folder contains maintenance and developer automation scripts.

- `gen_proto.sh`: regenerate Go and TypeScript protobuf bindings.
- `import_country_borders.py`: build canonical global and theater border
  GeoJSON from `ne_10m_admin_0_countries/ne_10m_admin_0_countries.shp`.
- `import_territorial_waters.py`: build canonical global and theater maritime
  GeoJSON from `World_12NM_v4_20231025_gpkg/eez_12nm_v4.gpkg`.
- `reorganize_unit_yamls.go`: regroup unit library files by domain and country of origin.

The raw shapefile / GeoPackage source folders are local preprocessing inputs
and should stay out of version control; the generated JSON in
`internal/geo/data/` is the tracked artifact.

Scripts here should be safe to rerun and should encode repo conventions instead of relying on one-off manual steps.
