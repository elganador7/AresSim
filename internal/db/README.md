# DB

This folder owns persistence and schema evolution for the backend.

- `schema.go`: canonical database schema and schema version.
- `manager.go`: storage mode selection and DB process lifecycle.
- `client.go`: DB client construction.
- `checkpoint.go`: scenario/sim snapshot persistence.

Any new field that must survive restart should be added here and then threaded through the app bridge and frontend stores. Keep schema changes explicit and bump the schema version when incompatible changes require a rebuild.
