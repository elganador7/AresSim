#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# gen_proto.sh
#
# Compiles all proto files in proto/ to:
#   Go  → internal/gen/engine/
#   TS  → frontend/src/proto/
#
# Prerequisites:
#   brew install buf
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   npm install -g @bufbuild/protoc-gen-es @bufbuild/buf
#
# Run from repo root:
#   ./scripts/gen_proto.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "→ Cleaning generated output..."
rm -rf internal/gen/engine
mkdir -p internal/gen/engine/v1
mkdir -p frontend/src/proto/engine/v1

echo "→ Running buf generate..."
buf generate proto

echo "→ Verifying Go output..."
go build ./internal/gen/...

echo "✓ Proto generation complete."
echo "  Go  → internal/gen/engine/v1/"
echo "  TS  → frontend/src/proto/engine/v1/"
