# Sim

This folder contains the core simulation engine.

- `adjudicator.go`: detection, target selection, engagement, and kill logic.
- `munitions.go`: in-flight weapon updates and arrival resolution.
- `mock.go`: event generation and integration glue for the mock sim loop.
- `*_test.go`: behavioral coverage for sim rules.

This is the highest-leverage area for gameplay changes. Keep rules here data-driven where possible, because scenario complexity is growing quickly.
