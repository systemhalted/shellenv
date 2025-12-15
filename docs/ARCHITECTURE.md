# shellenv Architecture

shellenv provides per-project shell sandboxes so scripts can be exercised against specific shells and option profiles without touching a user's login shell.

## Layout and data
- **Global home** (`SHELLENV_HOME`, default `~/.shellenv`): created by `shellenv init` with `installs/`, `shims/`, `cache/`, and `tmp/`. A `.initialized` marker is written as part of setup.
- **Project envs** (`./.shellenv/<env>`): contain `metadata.json`, a `bin/` directory for project-local tools, and optional helper files (e.g., `activate.sh`). `shellenv create` scaffolds this structure; `shellenv use` records the current env in `./.shellenv/current`.
- **Profiles**: sourced shell options under `profiles/` (built-ins: `strict`, `posix`, `interactive`). Overridable via `SHELLENV_PROFILES` or `./profiles/`.

## Key packages and libraries
- **Cobra (github.com/spf13/cobra)**: command-line framework used to define commands, flags, help text, and dispatch (`rootCmd` + subcommands under `internal/cli`). Each command’s `RunE` returns an error so non-zero exits propagate.
- `cmd/shellenv`: CLI entrypoint calling `cli.Execute()` to run the Cobra root command.
- `internal/cli/*`: Cobra command implementations (`init`, `create`, `activate`, `exec`, `install`, etc.).
- `internal/env`: resolves and prepares `SHELLENV_HOME`.
- `internal/project`: per-project metadata reading/writing, env listing, and current-env tracking.
- `internal/shell`: activation snippet generation and profile resolution.

## Command flows
- **init**: ensures `SHELLENV_HOME` exists, prints PATH instructions, writes `.initialized`.
- **create**: writes `metadata.json` (name, shell, profile, tools placeholder) and ensures `bin/` exists under `./.shellenv/<env>`.
- **activate**: picks an env (arg → `./.shellenv/current` → `default`), verifies it exists, resolves the profile (`SHELLENV_PROFILES` → `./profiles` → alongside the binary), and prints shell code that sets `SHELLENV_ACTIVE=1`, `SHELLENV_ENV_NAME`, prepends `bin/` to `PATH`, and prefixes the prompt. Fish activation skips profile sourcing.
- **exec**: uses the same env selection as `activate`, builds a child env with `bin/` prepended and `SHELLENV_*` vars set, resolves the command path against that PATH (so project-local tools win), then runs it without requiring interactive activation.
- **install/uninstall/versions**: placeholder runtime managers that create/remove directories under `$SHELLENV_HOME/installs/`.
- **which**: resolves a binary preferring the env `bin/` folder.
- **destroy/list**: remove or enumerate project envs under `./.shellenv`.

### Command selection details
- **Env selection**: `activate`/`exec` pick env → CLI arg > `./.shellenv/current` > `default`.
- **Profile lookup**: `SHELLENV_PROFILES` > `./profiles/<name>.sh` > alongside the built binary (`dist/.../profiles`).
- **PATH resolution in exec**: `exec` prepends `<env>/bin` to PATH and resolves the command path manually, honoring the hardened Go 1.19+ rule that ignores the current directory. This ensures project-local shims/binaries run even when the OS PATH search would skip `./`.
- **Fish vs POSIX**: activation for fish skips profile sourcing (no POSIX `source`) and uses fish env syntax; bash/zsh/posix shells get `SHELLENV_*`, prompt prefix, and optional profile sourcing.

## Environment variables
- `SHELLENV_HOME`: global state root (default `~/.shellenv`).
- `SHELLENV_PROFILES`: optional directory override for profile lookup.
- `SHELLENV_ACTIVE`: set to `1` when an env is activated or when using `exec`.
- `SHELLENV_ENV_NAME`: active env name for prompt and debugging.

## Testing
- Unit tests: `make test`.
- Integration tests (require `bats`): `SHELLENV_HOME=$(mktemp -d) bats -r test/integration`.
- A fresh `dist/shellenv` binary is expected for integration runs (`make build`).
