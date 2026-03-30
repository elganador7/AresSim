# 3D Asset Pipeline

This project now supports a mixed unit-visual pipeline:

- far zoom: billboard icon
- close zoom: 3D model when available
- fallback: close-range proxy geometry when no model asset exists

## Current layout

- raw third-party model drops:
  - `frontend/public/assets/models/raw/<asset-id>/`
- frontend model manifest:
  - `frontend/src/components/cesium/modelAssets.ts`
- visual profile selection:
  - `frontend/src/components/cesium/visualModels.ts`
- backend visual model id inference:
  - `app_bridge.go`

## How model selection works

1. Backend `ListUnitDefinitions()` emits `visual_model_id`.
2. Frontend stores that in Cesium definition metadata.
3. Renderer checks `modelAssets.ts`.
4. If a real model exists, it renders the model close-in.
5. Otherwise it uses a close-in proxy shape.
6. Billboards remain at distance for readability.

## Naming convention

Use one stable visual ID per asset family:

- `ddg51`
- `f35`
- `f22`
- `f15`
- `aew`
- `tanker`
- `patriot`
- `s300`
- `bavar373`
- `frigate`
- `corvette`
- `submarine`

## Recommended asset format

Preferred:

- `.glb`

Also supported:

- `.gltf` with adjacent `.bin` and texture files

`glb` is preferred because it is easier to manage and ship as a single file.

## Normalization rules

Before promoting a raw asset into the main manifest:

1. Confirm license and attribution terms.
2. Convert to `.glb` when practical.
3. Normalize forward orientation.
4. Normalize scale to plausible real-world proportions.
5. Reduce polygon count if the asset is too heavy.
6. Verify it renders correctly in Cesium.

## Attribution

When using CC-BY assets, keep attribution in:

- `frontend/public/assets/models/raw/<asset-id>/license.txt`

and mirror any shipped credits in release notes or an in-app credits section later.

## Current real asset

- `ddg51`
  - source folder: `frontend/public/assets/models/raw/ddg51/`
  - shipped file: `ddg51.glb`
  - currently mapped to Arleigh Burke-class destroyers
