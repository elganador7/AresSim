#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[slow] rebuild oil render cache"
go run ./cmd/build-oil-renderable-cache

echo "[slow] proving grounds"
go run ./cmd/proving-ground --scenario all --trials 10
