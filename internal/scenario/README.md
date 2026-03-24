# Scenario

This folder contains built-in scenarios and scenario seed data.

- `builtins.go`: list of built-in scenarios seeded at startup.
- `default.go`: generic default scenario helpers.
- `iran_coalition_war_skeleton.go`: current major review scenario for the Iran war work.
- `proving_ground.go`: small calibration scenarios and expected benchmark metadata.
- `weapons.go`: shared weapon catalog used by the sim and default content.

Add scenario-specific force layouts here, but keep reusable platform and weapon data in `internal/library/` and `weapons.go`.
