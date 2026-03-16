# Internal

This directory contains the Go backend for the simulator.

- `db/`: storage setup, schema, checkpoints, and DB process management.
- `library/`: YAML-backed unit definition loading and normalization.
- `scenario/`: built-in scenarios and shared scenario seed data such as weapons.
- `sim/`: detection, engagement, munition, and tick-resolution logic.
- `gen/`: generated protobuf bindings for the Go backend. Do not edit by hand.

When changing behavior, prefer touching `sim/` and `scenario/` first, then wire new fields through `library/`, `db/`, and the app bridge.
