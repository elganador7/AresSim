#!/usr/bin/env python3
"""Normalize oil-relevant GOGI layers into database-friendly NDJSON exports."""

from __future__ import annotations

import argparse
import json
from functools import lru_cache
from pathlib import Path
from typing import Any

import fiona
from fiona.errors import TransformError
from fiona.transform import transform_geom
from pyproj import Transformer


SOURCE_CANDIDATES = [
    "data/GOGI_V10_3_1.gdb",
    "data/GOGI_V10_3.gdb",
    "data/GOGI_V10_3SHP",
    "data/gogi_v10_3_1shp",
    "data/GOGI_V10_2.gdb",
    "data/GOGI_V10_2SHP",
]

SPATIAL_LAYER_CONFIG = {
    "Fields": {"category": "field", "include_outline": False},
    "Refineries": {"category": "refinery", "include_outline": False},
    "Ports": {"category": "port", "include_outline": False},
    "Storage": {"category": "storage", "include_outline": False},
    "Underground_Storage": {"category": "underground_storage", "include_outline": False},
    "Processing_Plants": {"category": "processing_plant", "include_outline": False},
    "LNG": {"category": "lng_terminal", "include_outline": False},
    "Platforms_and_Well_Pads": {"category": "platform_or_well_pad", "include_outline": False},
    "Stations": {"category": "station", "include_outline": False},
    "Power_Plants": {"category": "power_plant", "include_outline": False},
    "Basins": {"category": "basin", "include_outline": False},
    "Well": {"category": "well", "include_outline": False},
    "Wells": {"category": "well", "include_outline": False},
    "Active_Wells": {"category": "active_well_grid", "include_outline": False},
    "Wells_Vector_Grid": {"category": "well_grid", "include_outline": False},
    "Pipelines": {"category": "pipeline", "include_outline": False},
}

CATALOG_PREFIX = "Data_Catalog_"


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Extract GOGI oil and gas layers into per-layer normalized NDJSON files."
    )
    parser.add_argument("--input", help="GOGI source path (.gdb or shapefile directory).")
    parser.add_argument(
        "--output-dir",
        default="data/gogi-normalized-v1",
        help="Directory for normalized outputs.",
    )
    parser.add_argument(
        "--max-line-points",
        type=int,
        default=32,
        help="Maximum vertices per line part in exported routes.",
    )
    parser.add_argument(
        "--max-polygon-points",
        type=int,
        default=32,
        help="Maximum vertices per polygon ring when outlines are included.",
    )
    parser.add_argument(
        "--include-field-outlines",
        action="store_true",
        help="Include simplified field polygon outlines.",
    )
    parser.add_argument(
        "--include-pipelines",
        action="store_true",
        help="Export GOGI pipelines too. Off by default because GEM is the primary pipeline source.",
    )
    parser.add_argument(
        "--include-raw",
        action="store_true",
        help="Include a trimmed raw property map on each exported row.",
    )
    parser.add_argument(
        "--layers",
        help="Comma-separated subset of spatial layers to export. Defaults to all supported layers.",
    )
    args = parser.parse_args()

    source = resolve_input(args.input)
    selected_layers = parse_selected_layers(args.layers)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    layers_dir = output_dir / "layers"
    catalogs_dir = output_dir / "catalogs"
    layers_dir.mkdir(parents=True, exist_ok=True)
    catalogs_dir.mkdir(parents=True, exist_ok=True)

    manifest = {
        "source": str(source),
        "layers": [],
        "catalogs": [],
        "counts": {
            "spatialRecords": 0,
            "catalogRows": 0,
        },
        "options": {
            "includeFieldOutlines": args.include_field_outlines,
            "includePipelines": args.include_pipelines,
            "includeRaw": args.include_raw,
            "maxLinePoints": args.max_line_points,
            "maxPolygonPoints": args.max_polygon_points,
        },
    }

    for layer_name in fiona.listlayers(str(source)):
        if layer_name.startswith(CATALOG_PREFIX):
            if selected_layers is not None:
                continue
            rows = export_catalog_layer(source, layer_name, catalogs_dir)
            manifest["catalogs"].append(
                {
                    "layer": layer_name,
                    "path": str(catalogs_dir / f"{slugify(layer_name)}.ndjson"),
                    "rowCount": rows,
                }
            )
            manifest["counts"]["catalogRows"] += rows
            continue

        config = SPATIAL_LAYER_CONFIG.get(layer_name)
        if config is None:
            continue
        if selected_layers is not None and layer_name not in selected_layers:
            continue
        if layer_name == "Pipelines" and not args.include_pipelines:
            continue

        print(f"exporting layer {layer_name}...")
        count, geometry = export_spatial_layer(
            source=source,
            layer_name=layer_name,
            category=config["category"],
            output_dir=layers_dir,
            max_line_points=args.max_line_points,
            max_polygon_points=args.max_polygon_points,
            include_field_outlines=args.include_field_outlines,
            include_raw=args.include_raw,
        )
        manifest["layers"].append(
            {
                "layer": layer_name,
                "category": config["category"],
                "geometry": geometry,
                "path": str(layers_dir / f"{slugify(layer_name)}.ndjson"),
                "recordCount": count,
            }
        )
        manifest["counts"]["spatialRecords"] += count

    (output_dir / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(
        f"normalized GOGI from {source} -> {output_dir} "
        f"({manifest['counts']['spatialRecords']} spatial records, {manifest['counts']['catalogRows']} catalog rows)"
    )
    return 0


def parse_selected_layers(raw: str | None) -> set[str] | None:
    if raw is None:
        return None
    names = {part.strip() for part in raw.split(",") if part.strip()}
    unknown = sorted(names - set(SPATIAL_LAYER_CONFIG))
    if unknown:
        raise SystemExit(f"unknown layer names: {', '.join(unknown)}")
    return names


def resolve_input(path: str | None) -> Path:
    if path:
        candidate = Path(path)
        if not candidate.exists():
            raise SystemExit(f"input path does not exist: {candidate}")
        return candidate
    for candidate_str in SOURCE_CANDIDATES:
        candidate = Path(candidate_str)
        if candidate.exists():
            return candidate
    raise SystemExit("no GOGI dataset found under data/")


def export_spatial_layer(
    source: Path,
    layer_name: str,
    category: str,
    output_dir: Path,
    max_line_points: int,
    max_polygon_points: int,
    include_field_outlines: bool,
    include_raw: bool,
) -> tuple[int, str | None]:
    output_path = output_dir / f"{slugify(layer_name)}.ndjson"
    count = 0
    geometry_type: str | None = None
    with fiona.open(str(source), layer=layer_name) as src, output_path.open("w", encoding="utf-8") as handle:
        geometry_type = src.schema.get("geometry")
        for feature in src:
            record = normalize_spatial_feature(
                source_path=str(source),
                layer_name=layer_name,
                category=category,
                feature=feature,
                src_crs=src.crs,
                max_line_points=max_line_points,
                max_polygon_points=max_polygon_points,
                include_field_outlines=include_field_outlines,
                include_raw=include_raw,
            )
            handle.write(json.dumps(record) + "\n")
            count += 1
    return count, geometry_type


def export_catalog_layer(source: Path, layer_name: str, output_dir: Path) -> int:
    output_path = output_dir / f"{slugify(layer_name)}.ndjson"
    count = 0
    with fiona.open(str(source), layer=layer_name) as src, output_path.open("w", encoding="utf-8") as handle:
        for feature in src:
            handle.write(
                json.dumps(
                    {
                        "source": "GOGI",
                        "sourcePath": str(source),
                        "sourceLayer": layer_name,
                        "row": normalize_properties(feature["properties"]),
                    }
                )
                + "\n"
            )
            count += 1
    return count


def normalize_spatial_feature(
    source_path: str,
    layer_name: str,
    category: str,
    feature: dict[str, Any],
    src_crs: Any,
    max_line_points: int,
    max_polygon_points: int,
    include_field_outlines: bool,
    include_raw: bool,
) -> dict[str, Any]:
    props = normalize_properties(feature["properties"])
    geom = to_wgs84_geometry(feature.get("geometry"), src_crs)
    geom_type = geom["type"] if geom else None
    bounds = geometry_bounds(geom)
    centroid = geometry_centroid(geom, bounds)

    record = {
        "source": "GOGI",
        "sourcePath": source_path,
        "sourceLayer": layer_name,
        "category": category,
        "sourceKey": first_present(props, ["MD_Fkey"]),
        "name": first_present(props, ["Facility_Name", "Facility_N", "Dataset"]),
        "country": first_present(props, ["MD_Country", "Country"]),
        "region": first_present(props, ["MD_Region", "Spatial_extent", "Spatial_Extent"]),
        "status": first_present(props, ["Status"]),
        "type": first_present(props, ["Type", "Category", "Catagory"]),
        "commodity": first_present(props, ["Commodity"]),
        "capacity": first_present(props, ["Capacity"]),
        "throughput": first_present(props, ["Throughput"]),
        "diameter": first_present(props, ["Diameter"]),
        "operator": first_present(props, ["Operator", "Data_Source_Owner"]),
        "onshoreOffshore": first_present(props, ["Onshore_Offshore", "Onshore_Of"]),
        "installationDate": first_present(props, ["Installation_Date", "Installati", "Date_Aquired", "Date_Acquired"]),
        "sourceURL": first_present(props, ["MD_Source", "Available_at__link_to_data_source_"]),
        "sourceDate": first_present(props, ["MD_Source_Date", "MD_Source_"]),
        "geometryType": geom_type,
        "centroid": centroid,
        "bounds": bounds,
    }

    if category == "pipeline":
        record["routes"] = line_routes(geom, max_line_points) if geom else []
    elif include_field_outlines and category == "field" and geom_type in {"Polygon", "MultiPolygon"}:
        record["outlineRings"] = polygon_rings(geom, max_polygon_points)

    if include_raw:
        record["raw"] = trim_raw_properties(props)
    return record


def to_wgs84_geometry(geometry: dict[str, Any] | None, src_crs: Any) -> dict[str, Any] | None:
    if not geometry:
        return None
    if not src_crs:
        return geometry
    if geometry.get("type") in {"Point", "3D Point"}:
        return transform_point_geometry(geometry, src_crs)
    try:
        return transform_geom(src_crs, "EPSG:4326", geometry, antimeridian_cutting=False)
    except TransformError:
        try:
            with fiona.Env(OGR_ENABLE_PARTIAL_REPROJECTION="TRUE"):
                return transform_geom(src_crs, "EPSG:4326", geometry, antimeridian_cutting=False)
        except TransformError:
            return None


@lru_cache(maxsize=16)
def get_transformer(src_crs_key: str) -> Transformer:
    return Transformer.from_crs(src_crs_key, "EPSG:4326", always_xy=True)


def transform_point_geometry(geometry: dict[str, Any], src_crs: Any) -> dict[str, Any] | None:
    coords = geometry.get("coordinates")
    if not coords or len(coords) < 2:
        return None
    try:
        transformer = get_transformer(str(src_crs))
        x, y = transformer.transform(coords[0], coords[1])
    except Exception:
        return None
    transformed = [round(x, 6), round(y, 6)]
    if len(coords) > 2:
        transformed.extend(coords[2:])
    return {"type": "Point", "coordinates": transformed}


def line_routes(geom: dict[str, Any], max_points: int) -> list[list[list[float]]]:
    if geom["type"] == "LineString":
        return [downsample_coords(geom["coordinates"], max_points)]
    if geom["type"] == "MultiLineString":
        return [downsample_coords(part, max_points) for part in geom["coordinates"]]
    return []


def polygon_rings(geom: dict[str, Any], max_points: int) -> list[list[list[float]]] | None:
    if geom["type"] == "Polygon":
        polygons = [geom["coordinates"]]
    elif geom["type"] == "MultiPolygon":
        polygons = geom["coordinates"]
    else:
        return None
    rings: list[list[list[float]]] = []
    for polygon in polygons:
        if polygon:
            rings.append(downsample_coords(polygon[0], max_points))
    return rings or None


def downsample_coords(coords: list[Any], max_points: int) -> list[list[float]]:
    if len(coords) <= max_points or max_points < 2:
        return [[round(coord[0], 6), round(coord[1], 6)] for coord in coords]
    last_idx = len(coords) - 1
    sampled: list[list[float]] = []
    for index in range(max_points):
        coord_index = int(index * last_idx / (max_points - 1))
        coord = coords[coord_index]
        sampled.append([round(coord[0], 6), round(coord[1], 6)])
    return sampled


def geometry_bounds(geom: dict[str, Any] | None) -> list[float] | None:
    if not geom:
        return None
    coords = list(iter_coords(geom["coordinates"]))
    if not coords:
        return None
    xs = [coord[0] for coord in coords]
    ys = [coord[1] for coord in coords]
    return [round(min(xs), 6), round(min(ys), 6), round(max(xs), 6), round(max(ys), 6)]


def geometry_centroid(geom: dict[str, Any] | None, bounds: list[float] | None) -> list[float] | None:
    if not geom:
        return None
    if geom["type"] in {"Point", "3D Point"}:
        coord = geom["coordinates"]
        return [round(coord[0], 6), round(coord[1], 6)]
    if bounds:
        return [round((bounds[0] + bounds[2]) / 2, 6), round((bounds[1] + bounds[3]) / 2, 6)]
    return None


def iter_coords(coords: Any):
    if not isinstance(coords, (list, tuple)):
        return
    if coords and isinstance(coords[0], (int, float)):
        yield coords
        return
    for value in coords:
        yield from iter_coords(value)


def normalize_properties(properties: Any) -> dict[str, Any]:
    if not hasattr(properties, "items"):
        return {}
    return {
        key: normalize_value(value)
        for key, value in properties.items()
        if value not in ("", None)
    }


def normalize_value(value: Any) -> Any:
    if hasattr(value, "isoformat"):
        return value.isoformat()
    return value


def trim_raw_properties(values: dict[str, Any]) -> dict[str, Any]:
    useful_keys = [
        "MD_Fkey",
        "MD_Country",
        "MD_Region",
        "Facility_Name",
        "Facility_N",
        "Status",
        "Type",
        "Commodity",
        "Capacity",
        "Throughput",
        "Diameter",
        "Operator",
        "Onshore_Offshore",
        "Onshore_Of",
        "Installation_Date",
        "Installati",
        "MD_Source",
        "MD_Source_Date",
        "MD_Source_",
        "Spat_Ranks",
        "Temp_Ranks",
        "Sour_Ranks",
        "Shape_Length",
        "Shape_Area",
        "Count",
    ]
    return {key: values[key] for key in useful_keys if key in values}


def first_present(values: dict[str, Any], keys: list[str]) -> Any:
    for key in keys:
        if key in values and values[key] not in ("", None):
            return values[key]
    return None


def slugify(value: str) -> str:
    return "".join(ch.lower() if ch.isalnum() else "_" for ch in value).strip("_")


if __name__ == "__main__":
    raise SystemExit(main())
