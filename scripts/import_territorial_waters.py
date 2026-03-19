#!/usr/bin/env python3

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import geopandas as gpd
from shapely.geometry import mapping


REPO_ROOT = Path(__file__).resolve().parents[1]
SOURCE_GPKG = REPO_ROOT / "World_12NM_v4_20231025_gpkg" / "eez_12nm_v4.gpkg"
SOURCE_LAYER = "eez_12nm_v4"

GLOBAL_OUTPUT = REPO_ROOT / "internal" / "geo" / "data" / "world_12nm.json"
THEATER_OUTPUT = REPO_ROOT / "internal" / "geo" / "data" / "theater_maritime.json"

THEATER_BBOX = (15.0, 10.0, 65.0, 45.0)
GLOBAL_SIMPLIFY_TOLERANCE = 0.05
THEATER_SIMPLIFY_TOLERANCE = 0.05
ROUND_DECIMALS = 5


def round_coords(value: Any) -> Any:
    if isinstance(value, (list, tuple)):
        return [round_coords(v) for v in value]
    if isinstance(value, float):
        return round(value, ROUND_DECIMALS)
    return value


def to_feature_collection(gdf: gpd.GeoDataFrame) -> dict[str, Any]:
    features: list[dict[str, Any]] = []
    for _, row in gdf.iterrows():
        geom = row.geometry
        if geom is None or geom.is_empty:
            continue
        features.append({
            "type": "Feature",
            "properties": {
                "owner": row["owner"],
                "zoneType": "territorial_sea",
            },
            "geometry": round_coords(mapping(geom)),
        })
    return {
        "type": "FeatureCollection",
        "features": features,
    }


def prepare_frame(gdf: gpd.GeoDataFrame, simplify_tolerance: float) -> gpd.GeoDataFrame:
    prepared = gdf.copy()
    prepared["owner"] = (
        prepared["ISO_TER1"]
        .fillna(prepared["ISO_SOV1"])
        .astype(str)
        .str.strip()
        .str.upper()
    )
    prepared = prepared[(prepared["owner"] != "") & (prepared["owner"] != "NAN")]
    prepared = prepared[["owner", "geometry"]]
    prepared["geometry"] = prepared.geometry.make_valid()
    if simplify_tolerance > 0:
        prepared["geometry"] = prepared.geometry.simplify(simplify_tolerance, preserve_topology=True)
    prepared = prepared[~prepared.geometry.is_empty]
    return prepared


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, separators=(",", ":")) + "\n", encoding="utf-8")


def main() -> None:
    if not SOURCE_GPKG.exists():
        raise SystemExit(f"Missing source GeoPackage: {SOURCE_GPKG}")

    global_gdf = gpd.read_file(SOURCE_GPKG, layer=SOURCE_LAYER)
    theater_gdf = gpd.read_file(SOURCE_GPKG, layer=SOURCE_LAYER, bbox=THEATER_BBOX)

    global_payload = to_feature_collection(prepare_frame(global_gdf, GLOBAL_SIMPLIFY_TOLERANCE))
    theater_payload = to_feature_collection(prepare_frame(theater_gdf, THEATER_SIMPLIFY_TOLERANCE))

    write_json(GLOBAL_OUTPUT, global_payload)
    write_json(THEATER_OUTPUT, theater_payload)

    print(f"Wrote {GLOBAL_OUTPUT.relative_to(REPO_ROOT)} with {len(global_payload['features'])} features")
    print(f"Wrote {THEATER_OUTPUT.relative_to(REPO_ROOT)} with {len(theater_payload['features'])} features")


if __name__ == "__main__":
    main()
