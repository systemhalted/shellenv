# Task Log

- Added `docs/ARCHITECTURE.md` outlining layout, key packages, command flows, and env variables; expanded with Cobra and exec resolution details.
- Updated `README.md` with links to documentation and a disclaimer (use at own risk, no warranties).
- Added MIT `LICENSE` to align with the warranty disclaimer.
- Added `ExecuteWithArgs` helper for CLI tests and covered `cmd/shellenv` with help/list tests.
- Recorded this task log in `docs/Task.md`.

Tests: `GOCACHE=$(pwd)/.gocache go test ./...`.
