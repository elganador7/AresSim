# Utils

This folder contains reusable frontend helpers.

- `formatters.ts`: user-facing text/number formatting.
- `geo.ts`: geographic math helpers.
- `unitBillboard.ts`: map icon and badge generation.

Keep utilities pure where possible. If a helper starts needing store access or Wails calls, it likely belongs somewhere else.

## Review Notes

- `countryRelationships.ts` is now intentionally narrow: normalization and lightweight country collection only. Keep relationship policy in backend preview/enforcement code rather than rebuilding it in frontend helpers.
- Access-control and path-legality helpers should stay out of this folder unless they are truly UI-only. The backend preview layer is now the authority for route/strike legality.
