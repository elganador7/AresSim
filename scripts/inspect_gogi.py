#!/usr/bin/env python3
"""Inspect GOGI ArcGIS/shapefile datasets and emit a compact JSON summary."""

from __future__ import annotations

import argparse
import csv
import json
from pathlib import Path
from typing import Any

import fiona


DEFAULT_GLOBS = ["data/GOGI*", "data/gogi*"]


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Inspect GOGI shapefile/geodatabase datasets and print a JSON summary."
    )
    parser.add_argument(
        "paths",
        nargs="*",
        help="Dataset paths to inspect. Defaults to data/GOGI* and data/gogi*.",
    )
    parser.add_argument(
        "--max-samples",
        type=int,
        default=3,
        help="Maximum sample features/rows to include per layer or CSV.",
    )
    parser.add_argument(
        "--output",
        help="Optional output path for the JSON summary.",
    )
    args = parser.parse_args()

    targets = expand_targets(args.paths)
    summary = {
        "datasets": [inspect_dataset(path, args.max_samples) for path in targets],
    }

    payload = json.dumps(summary, indent=2)
    if args.output:
        Path(args.output).write_text(payload + "\n", encoding="utf-8")
    else:
        print(payload)
    return 0


def expand_targets(paths: list[str]) -> list[Path]:
    if paths:
        return [Path(p) for p in paths]

    seen: set[Path] = set()
    targets: list[Path] = []
    for pattern in DEFAULT_GLOBS:
        for path in sorted(Path(".").glob(pattern)):
            if path not in seen:
                seen.add(path)
                targets.append(path)
    return targets


def inspect_dataset(path: Path, max_samples: int) -> dict[str, Any]:
    dataset: dict[str, Any] = {
        "path": str(path),
        "exists": path.exists(),
        "type": detect_dataset_type(path),
    }
    if not path.exists():
        dataset["error"] = "path does not exist"
        return dataset

    try:
        if dataset["type"] in {"filegdb", "shapefile_dir"}:
            dataset["layers"] = inspect_vector_container(path, max_samples)
            dataset["catalog_csvs"] = inspect_catalog_csvs(path, max_samples)
        elif dataset["type"] == "shapefile":
            dataset["layers"] = [inspect_layer(path, None, max_samples)]
        elif dataset["type"] == "csv":
            dataset["csv"] = inspect_csv(path, max_samples)
        else:
            dataset["children"] = [child.name for child in sorted(path.iterdir())[:50]] if path.is_dir() else []
    except Exception as exc:  # pragma: no cover - defensive for opaque GIS errors
        dataset["error"] = f"{type(exc).__name__}: {exc}"
    return dataset


def detect_dataset_type(path: Path) -> str:
    if path.suffix.lower() == ".gdb" and path.is_dir():
        return "filegdb"
    if path.suffix.lower() == ".shp":
        return "shapefile"
    if path.suffix.lower() == ".csv":
        return "csv"
    if path.is_dir() and any(path.glob("*.shp")):
        return "shapefile_dir"
    if path.is_dir():
        return "directory"
    return "file"


def inspect_vector_container(path: Path, max_samples: int) -> list[dict[str, Any]]:
    layers: list[dict[str, Any]] = []
    for layer_name in fiona.listlayers(str(path)):
        layers.append(inspect_layer(path, layer_name, max_samples))
    return layers


def inspect_layer(path: Path, layer_name: str | None, max_samples: int) -> dict[str, Any]:
    kwargs = {}
    if layer_name is not None:
        kwargs["layer"] = layer_name

    with fiona.open(str(path), **kwargs) as src:
        samples: list[dict[str, Any]] = []
        try:
            feature_count = len(src)
        except Exception:
            feature_count = None
        for index, feature in enumerate(src):
            if index >= max_samples:
                break
            props = normalize_properties(feature["properties"])
            samples.append(props)
        bounds = None
        try:
            if src.bounds:
                bounds = list(src.bounds)
        except Exception:
            bounds = None

        return {
            "layer": layer_name or path.stem,
            "driver": src.driver,
            "crs": str(src.crs),
            "geometry": src.schema.get("geometry"),
            "featureCount": feature_count,
            "fields": src.schema.get("properties", {}),
            "bounds": bounds,
            "sampleProperties": samples,
        }


def inspect_catalog_csvs(path: Path, max_samples: int) -> list[dict[str, Any]]:
    csvs = sorted(path.glob("Data_Catalog*.csv"))
    return [inspect_csv(csv_path, max_samples) for csv_path in csvs]


def inspect_csv(path: Path, max_samples: int) -> dict[str, Any]:
    row_count = 0
    samples: list[dict[str, Any]] = []
    fieldnames: list[str] = []

    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        fieldnames = reader.fieldnames or []
        for row in reader:
            row_count += 1
            if len(samples) < max_samples:
                samples.append(
                    {
                        key: value
                        for key, value in row.items()
                        if value not in ("", None)
                    }
                )

    return {
        "path": str(path),
        "rowCount": row_count,
        "fields": fieldnames,
        "sampleRows": samples,
    }


def normalize_properties(properties: Any) -> dict[str, Any]:
    if hasattr(properties, "items"):
        items = properties.items()
    else:
        items = []
    return {key: value for key, value in items if value not in (None, "")}

if __name__ == "__main__":
    raise SystemExit(main())
