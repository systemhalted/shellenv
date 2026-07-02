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
- Expanded `docs/ARCHITECTURE.md` with an honest Isolation Model (PATH-shimming, boundary table, intended use vs non-use) and a Limitations section enumerating the P0/P1/P2 gaps.
- Added `docs/DESIGN.md` capturing eight design decisions (rationale/trade-offs/status) and a gap-closing Roadmap (R1–R6); linked it from `README.md`.
- R1: `exec` now isolates `HOME`/`TMPDIR`/`XDG_*` to a per-env sandbox (`./.shellenv/<env>/home/`); added `project.SandboxHomeDir` and an `upsertEnv` helper.
- R2: added `exec --profile` (opt-in), which sources the env's declared profile through the declared shell (`profileShell`) before running the command.
- Added exec tests for HOME/TMPDIR isolation and profile sourcing; updated ARCHITECTURE/DESIGN/README to match.
- R2 polish: silenced Cobra's usage/duplicate-error dump (`SilenceUsage`/`SilenceErrors`); `exec` now propagates a child's real exit code via an `exitError` type instead of always exiting 1, and runtime errors print a single clean line.
- Added `--container <image>` flag to `shellenv exec` to run commands inside Docker or Podman containers. Implemented TTY detection, volume mounts for the workspace, working directory mapping, environment variable forwarding, and entrypoint wrapping via `sh -c`. Added unit and integration tests, and updated documentation.

Tests: `GOCACHE=$(pwd)/.gocache go test ./...`.
