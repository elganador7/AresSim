#!/usr/bin/env python3

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import geopandas as gpd
from shapely.geometry import mapping


REPO_ROOT = Path(__file__).resolve().parents[1]
SOURCE_SHP = REPO_ROOT / "ne_10m_admin_0_countries" / "ne_10m_admin_0_countries.shp"

GLOBAL_OUTPUT = REPO_ROOT / "internal" / "geo" / "data" / "world_borders.json"
THEATER_OUTPUT = REPO_ROOT / "internal" / "geo" / "data" / "theater_borders.json"

THEATER_BBOX = (15.0, 10.0, 65.0, 45.0)
GLOBAL_SIMPLIFY_TOLERANCE = 0.03
THEATER_SIMPLIFY_TOLERANCE = 0.02
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
                "iso3": row["iso3"],
                "name": row["name"],
            },
            "geometry": round_coords(mapping(geom)),
        })
    return {
        "type": "FeatureCollection",
        "features": features,
    }


def prepare_frame(gdf: gpd.GeoDataFrame, simplify_tolerance: float) -> gpd.GeoDataFrame:
    prepared = gdf.copy()
    prepared["iso3"] = prepared["ADM0_A3"].astype(str).str.strip().str.upper()
    prepared["name"] = prepared["ADMIN"].astype(str).str.strip()
    prepared = prepared[(prepared["iso3"] != "") & (prepared["iso3"] != "NAN")]
    prepared = prepared[["iso3", "name", "geometry"]]
    prepared["geometry"] = prepared.geometry.make_valid()
    if simplify_tolerance > 0:
        prepared["geometry"] = prepared.geometry.simplify(simplify_tolerance, preserve_topology=True)
    prepared = prepared[~prepared.geometry.is_empty]
    return prepared


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, separators=(",", ":")) + "\n", encoding="utf-8")


def main() -> None:
    if not SOURCE_SHP.exists():
        raise SystemExit(f"Missing source shapefile: {SOURCE_SHP}")

    global_gdf = gpd.read_file(SOURCE_SHP)
    theater_gdf = gpd.read_file(SOURCE_SHP, bbox=THEATER_BBOX)

    global_payload = to_feature_collection(prepare_frame(global_gdf, GLOBAL_SIMPLIFY_TOLERANCE))
    theater_payload = to_feature_collection(prepare_frame(theater_gdf, THEATER_SIMPLIFY_TOLERANCE))

    write_json(GLOBAL_OUTPUT, global_payload)
    write_json(THEATER_OUTPUT, theater_payload)

    print(f"Wrote {GLOBAL_OUTPUT.relative_to(REPO_ROOT)} with {len(global_payload['features'])} features")
    print(f"Wrote {THEATER_OUTPUT.relative_to(REPO_ROOT)} with {len(theater_payload['features'])} features")


if __name__ == "__main__":
    main()
