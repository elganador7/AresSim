#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[fast] backend unit packages"
go test \
  ./internal/db/... \
  ./internal/geo \
  ./internal/library \
  ./internal/oilnet/... \
  ./internal/routing \
  ./internal/scenario \
  ./internal/sim \
  -count=1

echo "[fast] frontend tests"
npm test --prefix frontend -- --run

echo "[fast] frontend build"
npm run build --prefix frontend
