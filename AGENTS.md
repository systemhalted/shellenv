# Repository Guidelines

This document is a quick guide for contributors working on the shellenv CLI. Keep changes focused, tested, and isolated so you can iterate without touching a user’s real shell setup.

## Project Structure & Module Organization
- `cmd/shellenv`: CLI entrypoint.
- `internal/cli`: Cobra commands (`init`, `create`, `activate`, `use`, etc.).
- `internal/env`: Resolves `$SHELLENV_HOME` or `~/.shellenv`; ensures subdirectories (`installs/`, `cache/`, `tmp/`); resolves declared shell runtimes under `installs/`.
- `internal/project`: Per-project metadata under `./.shellenv/<env>/`.
- `internal/shell`: Activation/profile handling.
- `profiles/`: Option presets (`strict.sh`, `posix.sh`, `interactive.sh`).
- `test/`: Integration tests in `test/integration`; Go unit tests live next to code in `internal/*`.

## Build, Test, and Development Commands
- `make build`: Compile to `dist/shellenv`.
- `make test`: Run all Go unit tests.
- `bats -r test/integration`: Run integration tests (requires `bats` on PATH; set `SHELLENV_HOME=$(mktemp -d)` to avoid touching your real home).
- `./run.sh`: Example workflow (build, init, create, activate, basic checks); useful as a smoke test.

## Coding Style & Naming Conventions
- Go 1.22; format with `gofmt` (CI expectation).
- Keep Cobra commands in `internal/cli/<command>.go`; flag names are kebab-case (`--shell`, `--profile`).
- Prefer explicit errors and short, imperative help text; keep public surface minimal by defaulting to `internal` packages.

## Testing Guidelines
- Unit tests: `_test.go` files next to implementation (`internal/env/home_test.go`, `internal/project/project_test.go`, etc.); favor table-driven cases.
- Integration: Add `*.bats` under `test/integration`; use `run` and `assert` helpers, and point `SHELLENV_HOME` to a temp directory.
- If adding new profiles or activation logic, cover both Go unit tests and a Bats round-trip (create → activate → echo marker vars).


## Documentation
- Primary docs live in README.md, CONTRIBUTING.md, docs/ARCHITECTURE.md, and docs/TASKS.md. Add new docs under docs/ only when needed; get maintainer buy-in and link them from README.md.
- Content boundaries: README.md = install/usage/troubleshooting for end users; CONTRIBUTING.md = contributor workflow and standards; docs/ARCHITECTURE.md = design, flows, sequences; docs/TASKS.md = AI task log for completed work; add release notes/changelog to docs/ when we start versioning.
- Update docs whenever behavior changes: new/changed commands, flags, env vars, profiles, activation logic, or config defaults; keep examples in sync.
- Style/ownership: short headings, concise sentences, consistent formatting; a reviewer must sign off doc updates that accompany behavior changes.
- Always update the necessary files after completion of work and before the code is committed and pushed to the remote repo.


## Commit & Pull Request Guidelines
- Use concise, imperative commit messages (e.g., `Add profile resolver guard`, `Tighten activation prompt formatting`). No history exists yet, so set the tone.
- PRs should describe behavior changes, include reproduction/verification steps (`make build`, `make test`, `bats -r test/integration`), and note any env vars or new flags.
- Add tests with new features or bug fixes; call out gaps explicitly if something cannot be covered.


## Security & Configuration Tips
- Develop with `SHELLENV_HOME` pointed at a throwaway directory to keep installs out of your main home.
- Avoid persisting real shells or tools in repo paths; prefer temporary dirs under `./.shellenv/<env>/` during tests.
