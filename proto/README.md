# Protos

This folder defines the shared data contracts between the Go backend and the TypeScript frontend.

- `engine/v1/`: simulator-specific protobuf schemas.
- `buf.yaml` and `buf.gen.yaml`: protobuf tooling configuration.

If you add fields needed by both backend and frontend, change the relevant proto here first and then run `./scripts/gen_proto.sh`.
