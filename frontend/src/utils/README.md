# Utils

This folder contains reusable frontend helpers.

- `formatters.ts`: user-facing text/number formatting.
- `geo.ts`: geographic math helpers.
- `unitBillboard.ts`: map icon and badge generation.

Keep utilities pure where possible. If a helper starts needing store access or Wails calls, it likely belongs somewhere else.
