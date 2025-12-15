# Contributing to shellenv

Keep changes small, tested, and contained to avoid touching your real shell setup. This project targets Go 1.22.

## Setup
- Point shellenv at a throwaway home: `export SHELLENV_HOME="$(mktemp -d)"` (or manually create an empty directory you control).
- If your system restricts the default Go cache, use a repo-local cache: `GOCACHE=$PWD/.cache/go-build`.

## Build and test
- Build: `make build` (produces `dist/shellenv`).
- Unit tests: `make test` (or `GOCACHE=$PWD/.cache/go-build go test ./...`).
- Integration tests (requires `bats`): `SHELLENV_HOME=$(mktemp -d) bats -r test/integration`.
- Use `./run.sh` as a smoke test; it builds, inits, creates, activates, and performs basic checks.

## Coding style
- Run `gofmt` on Go code; Go version 1.22.
- Cobra commands live in `internal/cli/<command>.go`; flag names are kebab-case (`--shell`, `--profile`).
- Prefer explicit errors and concise help text; default to `internal` visibility unless needed externally.
- Keep public surface minimal and avoid touching user shells outside the project or `SHELLENV_HOME`.

## Documentation
- Update docs when behavior changes (commands, flags, env vars, profiles, activation logic, defaults).
- Primary docs: `README.md` (user install/usage), `CONTRIBUTING.md` (workflow/standards), `docs/ARCHITECTURE.md` (design/flows), `docs/Task.md` (AI task log).
- Add new files under `docs/` only when needed and link them from `README.md`.

## Commits and PRs
- Use concise, imperative commit messages (e.g., `Add profile resolver guard`).
- In PRs, include verification steps (`make build`, `make test`, `bats -r test/integration`) and note any new env vars or flags.
- Call out testing gaps if something cannot be covered.

## Safety
- Work inside a project directory; envs live in `./.shellenv/<env>`.
- Avoid persisting real shells or tools in repo paths; prefer temporary dirs under `./.shellenv/<env>/` during tests.
