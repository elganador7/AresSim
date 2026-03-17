# Sim

This folder contains the core simulation engine.

- `adjudicator.go`: detection, target selection, engagement, and kill logic.
- `munitions.go`: in-flight weapon updates and arrival resolution.
- `mock.go`: event generation and integration glue for the mock sim loop.
- `*_test.go`: behavioral coverage for sim rules.

This is the highest-leverage area for gameplay changes. Keep rules here data-driven where possible, because scenario complexity is growing quickly.

## Review Notes

- `mock.go`: `MockLoop` receives `RelationshipRules` as a startup snapshot. Runtime updates from `SetCountryRelationship()` mutate the scenario record, but the running sim keeps the old rule map until the loop is restarted. That makes live intel-sharing and airspace toggles appear to work in the UI while detection and behavior logic still use stale permissions.
- `mock.go`: attack auto-routing only seeds a waypoint once, then `hasExplicitMoveOrder()` suppresses further recalculation. If the assigned target moves significantly while the shooter is still en route, the attack path goes stale instead of continuing to reposition against the target.
- `app_sim.go`: clearing an attack task removes `AttackOrder` but leaves any previously auto-generated `MoveOrder` in place. That lets a unit continue flying the old strike ingress after the user thinks the task has been canceled.
- `adjudicator.go` and `mock.go`: engagement, movement reaction, and order-following logic are now tightly coupled across large functions. The next cleanup pass should split order execution, autonomous behavior, and effects resolution into smaller units before more doctrine features land.
