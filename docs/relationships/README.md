# Country Relationship Model

This document tracks the planned scenario-level diplomacy and access model for multi-country wars such as the Iran conflict.

## Goal
- Model country-to-country permissions directly.
- Do not derive behavior from broad coalition membership.
- Keep the first version simple enough to drive routing, tasking, and shared awareness.

## Core Concepts
- `team_id`: the owning country for a unit, e.g. `ISR`, `USA`, `IRN`.
- `coalition_id`: broad wartime alignment used by current combat logic.
- `employment_role`: whether a unit is `offensive`, `defensive`, or `dual_use`.
- `relationship matrix`: scenario-scoped country-to-country permissions.

## Phase 1 Relationship Fields
Each scenario should carry bilateral records keyed by `from_country -> to_country` with:

- `share_intel: bool`
- `airspace_transit_allowed: bool`
- `airspace_strike_allowed: bool`
- `defensive_positioning_allowed: bool`

These are intentionally binary for the first pass. More detailed quality or treaty logic can be layered on later.

## Intended Semantics
- `share_intel=true`: the receiving country may consume the other country’s shared detection picture.
- `airspace_transit_allowed=false`: no routing through that country’s airspace.
- `airspace_strike_allowed=false`: offensive attacks cannot be launched from or conducted through that country’s airspace.
- `defensive_positioning_allowed=true`: defensive systems may be placed or operate there even if offensive actions are blocked.

## Unit Role Rules
- `offensive`: ballistic missile units, strike-only systems, other deliberately offensive assets.
- `defensive`: SAM batteries, point defense systems, homeland-defense-only assets.
- `dual_use`: most fighters, ships, and general combat platforms.

These roles will be used to decide whether a unit can be positioned in a country and whether it can receive offensive orders from that territory.

## Planned Implementation Order
1. Add scenario relationship records and `employment_role` to data models.
2. Thread them through proto, DB, YAML, editor state, and runtime state.
3. Enforce country-specific intel sharing.
4. Enforce route legality and strike legality based on airspace permissions.
5. Enforce defensive-positioning checks.
6. Add editor UI for country-pair relationship editing.
7. Add runtime toggles for intel sharing later.

## Explicit Non-Goals For Phase 1
- No coalition-wide inheritance.
- No graded intel quality.
- No base-level permission model.
- No jamming, degraded comms, or treaty automation.
