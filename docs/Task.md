# Task Log

- Added `docs/ARCHITECTURE.md` outlining layout, key packages, command flows, and env variables; expanded with Cobra and exec resolution details.
- Updated `README.md` with links to documentation and a disclaimer (use at own risk, no warranties).
- Added MIT `LICENSE` to align with the warranty disclaimer.
- Added `ExecuteWithArgs` helper for CLI tests and covered `cmd/shellenv` with help/list tests.
- Added broad CLI unit coverage, gitignore for build artifacts, and documented running `go test` with a repo-local cache when permissions are restricted.
- Documented a short, step-by-step snippet for using shellenv inside a user's project directory (create/use/activate/exec).
- Expanded README usage section with safety notes explaining what each command does and that it stays within the project/temp home.
- Clarified `SHELLENV_HOME` setup with both mktemp-based and manual directory options for environments without `mktemp`.
- Added `CONTRIBUTING.md` with contributor workflow, testing commands, coding style, doc ownership, and safety guidance; linked from README.
- Recorded this task log in `docs/Task.md`.

Tests: `GOCACHE=$(pwd)/.gocache go test ./...`.
