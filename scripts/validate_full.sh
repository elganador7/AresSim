#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

"$ROOT_DIR/scripts/validate_fast.sh"

echo "[full] backend full suite"
go test ./... -count=1
