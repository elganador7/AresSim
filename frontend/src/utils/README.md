# Utils

This folder contains reusable frontend helpers.

- `formatters.ts`: user-facing text/number formatting.
- `geo.ts`: geographic math helpers.
- `unitBillboard.ts`: map icon and badge generation.

Keep utilities pure where possible. If a helper starts needing store access or Wails calls, it likely belongs somewhere else.

## Review Notes

- `countryRelationships.ts` and `theaterCountries.ts` now mirror logic that also exists in `internal/sim/relationships.go` and `internal/sim/theater_countries.go`. That duplication is currently intentional for UI explainability, but it is also a drift risk: transit/strike validation can disagree between frontend and backend unless both sides are updated together.
- Access-control and path-legality helpers are becoming policy code, not just formatting/math helpers. If they keep growing, they should move into a clearer shared “rules” layer rather than staying mixed with lightweight view utilities.
