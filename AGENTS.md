# Repository Guidelines

## Project Structure & Module Organization
`main.go` and `app.go` bootstrap the Wails desktop app and expose backend methods to the frontend. Core Go packages live under `internal/`: simulation logic in `internal/sim`, persistence in `internal/db`, scenario defaults in `internal/scenario`, and embedded library data in `internal/library`. The React/Vite UI lives in `frontend/src`, with editor components under `frontend/src/components/editor` and Zustand stores under `frontend/src/store`. Protobuf sources are in `proto/engine/v1`; generated Go and TypeScript outputs land in `internal/gen/engine/v1` and `frontend/src/proto/engine/v1`.

## Build, Test, and Development Commands
Run `wails dev` from the repo root for the full desktop app with live frontend reload. Use `go test ./...` to run backend tests and `npm test --prefix frontend` for Vitest. Build the frontend only with `npm run build --prefix frontend`. Regenerate protobuf bindings with `./scripts/gen_proto.sh`; this refreshes both Go and TS generated code. If dependencies are missing, install frontend packages with `npm install --prefix frontend`.

## Coding Style & Naming Conventions
Follow standard Go formatting with `gofmt`; Go files use tabs and exported identifiers use `CamelCase`. Keep packages focused and colocate tests with the code they cover. Frontend code is TypeScript with React function components; components use `PascalCase` filenames (`ScenarioEditor.tsx`), stores and utilities use `camelCase` (`simStore.ts`, `unitBillboard.ts`). Prefer clear comments only where the control flow or simulation rule is non-obvious.

## Testing Guidelines
Backend tests use Go’s `testing` package and are named `*_test.go`, for example `internal/sim/adjudicator_test.go`. Frontend tests use Vitest with `happy-dom`; place them beside the module as `*.test.ts`. Add regression tests for simulation behavior, store transitions, and bridge logic when changing those areas. Run both `go test ./...` and `npm test --prefix frontend` before opening a PR.

## Commit & Pull Request Guidelines
Recent history mixes short summaries with conventional commits, but `feat:` and scoped messages such as `refactor(editor): ...` are the clearest pattern. Prefer imperative commit subjects under 72 characters and include scope when useful. PRs should describe the behavior change, list the commands you ran, link related issues, and include screenshots or short recordings for UI/editor changes.

## Generated Code & Local Data
Do not hand-edit generated files in `internal/gen/engine/v1`, `frontend/src/proto/engine/v1`, or `frontend/wailsjs`. Rebuild them from source definitions instead. SurrealDB is started by the app at runtime; local app data and user libraries are managed through the backend, so avoid committing machine-specific generated state.
