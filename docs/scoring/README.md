# Scoring and War-Cost Model

The simulator's long-term score is intended to express the cost of war in
dollars, including both materiel and human loss.

## Current model

Unit definitions may author:

- `authorized_personnel`
- `replacement_cost_usd`
- `strategic_value_usd`
- `economic_value_usd`

If these are omitted, the loader applies benchmark defaults by
`asset_class`, `domain`, and `general_type`.

Live team score currently sums:

- `replacement_loss_usd`
- `strategic_loss_usd`
- `economic_loss_usd`
- `human_loss_usd`

`total_loss_usd` is the sum of all four.

## Human loss

Human loss is monetized with a Value of Statistical Life (VSL) benchmark.

Current default:

- `13.7M USD` per life

Source:

- U.S. Department of Transportation, current VSL estimate for analyses using
  a base year of 2024:
  https://www.transportation.gov/office-policy/transportation-policy/revised-departmental-guidance-on-valuation-of-a-statistical-life-in-economic-analysis

This is intentionally a public-policy valuation, not a compensation table.
The point is to force the score to reflect human loss in the same unit as
materiel loss.

## Asset benchmarks

The default replacement values are not intended to be exact per-platform
acquisition estimates. They are benchmarked to public U.S. program and
budget references, then generalized by class so the model works across many
countries and systems.

Reference anchors used for the current defaults include:

- F-35 program acquisition and unit-cost context:
  https://www.congress.gov/crs-product/R48304
- DDG-51 procurement cost context:
  https://www.congress.gov/crs-products/product/pdf/RL/RL32109/260
- Ford-class carrier procurement cost context:
  https://www.congress.gov/crs-product/RS20643
- Virginia-class submarine procurement cost context:
  https://www.congress.gov/crs-product/RL32418
- PATRIOT and THAAD budget context:
  https://www.congress.gov/crs-product/IN12447
  https://www.congress.gov/crs-product/IF12297

These defaults should be treated as baseline class valuations, not final truth.
For important scenario assets, authored values in YAML should override the
defaults.

## Known limitations

- The score is currently team-incurred loss, not attacker-attributed damage.
- Human loss currently uses authorized personnel and damage-state casualty
  fractions, not detailed casualty simulation.
- Civilian casualties outside directly modeled infrastructure staffing are not
  yet included.
- Economic disruption is represented by asset value loss, not a broader macro
  model.

## Next steps

- Author explicit valuations for the key Iran-war scenario assets.
- Add attacker attribution so the sim can report both `loss suffered` and
  `cost imposed`.
- Replace generic casualty fractions with asset-specific fatality / injury
  assumptions where scenario fidelity matters.
