# Sim

This folder contains the core simulation engine.

- `adjudicator.go`: detection, target selection, engagement, and kill logic.
- `munitions.go`: in-flight weapon updates and arrival resolution.
- `mock.go`: event generation and integration glue for the mock sim loop.
- `*_test.go`: behavioral coverage for sim rules.

This is the highest-leverage area for gameplay changes. Keep rules here data-driven where possible, because scenario complexity is growing quickly.

## Review Notes

- Resolved: `MockLoop` now reads relationship rules dynamically each tick, so live relationship toggles affect detection sharing and airspace behavior without requiring a loop restart.
- Resolved: attack auto-routing now distinguishes auto-generated ingress routes from manual routes, so moving targets can trigger replans without overwriting user-authored paths.
- Resolved: clearing an attack task now clears only the auto-generated attack ingress route, while preserving manual routes.
- Remaining cleanup: `adjudicator.go` and `mock.go` still carry too much coupled doctrine logic in large functions. The next cleanup pass should split order execution, autonomous behavior, and effects resolution into smaller units before more doctrine features land.
