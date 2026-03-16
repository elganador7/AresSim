# Bridge

This folder translates backend Wails events into frontend store updates.

- `bridge.ts`: event registration, protobuf decode helpers, and store dispatch.
- `bridge.test.ts`: lightweight coverage for bridge behavior.

Keep this layer thin. Decode and route data here, but keep presentation logic in components and durable state transitions in the stores.
