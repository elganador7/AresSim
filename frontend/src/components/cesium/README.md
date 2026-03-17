# Cesium Helpers

This folder contains helper modules for the live Cesium map.

- `helpers.ts`: shared visibility, team, route-legality, cursor, and entity helper logic.

Keep these modules focused on map-specific policy and rendering helpers. If a helper starts owning sim state or UI workflow directly, it likely belongs back in `store/` or a dedicated HUD/editor component instead.
