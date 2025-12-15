# shellenv

Per-project shell sandboxes for testing scripts against specific shells and option profiles without polluting your real shell setup.

## Intent
- Treat shells like runtimes: declare the shell version and profile a project expects, and run commands inside that sandbox.
- Keep experiments contained: environments live under `./.shellenv/<name>` and `SHELLENV_HOME` (default `~/.shellenv`), avoiding edits to your login shell.
- Make cross-shell QA easy: quickly swap between `bash`, `zsh`, `fish`, or POSIX-style profiles to catch portability issues early.

## Disclaimer
This project is provided as-is with no warranties; use at your own risk. See `LICENSE` for details.

## Prerequisites
- Go 1.22+ and `make` on your PATH.
- Integration tests: `bats` available (e.g., `brew install bats-core`, `apk add bats`, or `npm install -g bats`).

## How it fits together
- **Global home**: `shellenv init` ensures `SHELLENV_HOME` exists with `installs/`, `shims/`, `cache/`, and `tmp/`. Add `"$SHELLENV_HOME/shims"` to `PATH` if you use shims.
- **Project envs**: `shellenv create` writes metadata and a `bin/` directory to `./.shellenv/<env>`. Activation sets `SHELLENV_ACTIVE=1`, `SHELLENV_ENV_NAME`, prepends `bin/` to `PATH`, and prefixes your prompt.
- **Profiles**: Built-ins live in `profiles/` (`strict`, `posix`, `interactive`). `shellenv activate` will source a profile if found via `SHELLENV_PROFILES`, `./profiles/`, or next to the binary.
- **Runtimes**: `shellenv install <shell>@<version>` and `shellenv uninstall …` manage placeholders under `$SHELLENV_HOME/installs/`; `shellenv versions` lists what’s there.

## Quick start
```bash
# Build (Go 1.22)
make build

# Initialize global home and get PATH instructions
./dist/shellenv init

# Inside a project directory
./dist/shellenv create --shell bash@5.2 --profile strict
eval "$(./dist/shellenv activate)"   # or: ./dist/shellenv exec -- <cmd>
echo "$SHELLENV_ENV_NAME"            # -> default
```

## Docs
- Architecture and flows: `docs/ARCHITECTURE.md`.
- Task notes and change log: `docs/Task.md`.

## Common commands
- `shellenv create [--name default] --shell <shell>@<ver> [--profile strict]`: scaffold a project env.
- `shellenv activate [<env>] [--shell-type bash|zsh|fish]`: print activation snippet to `eval`.
- `shellenv exec [<env>] -- <cmd> [args]`: run a command without interactive activation.
- `shellenv use <env>` / `shellenv list` / `shellenv destroy <env>`: choose, inspect, or remove project envs.
- `shellenv install <shell>@<ver>` / `shellenv uninstall …` / `shellenv versions`: manage declared runtimes (placeholder installers today).
- `shellenv which <binary>`: resolve a tool inside the active env; `shellenv doctor`: quick health check.

## Testing and dev notes
- Unit tests: `make test`.
- Integration tests (require `bats`): `SHELLENV_HOME=$(mktemp -d) bats -r test/integration`.
- If your environment restricts writing to the default Go cache, run unit tests with a repo-local cache: `GOCACHE=$PWD/.cache/go-build go test ./...`.
- Keep experiments isolated by pointing `SHELLENV_HOME` at a temp directory when hacking on the tool.

## Example workflow
```bash
make build
./dist/shellenv init
./dist/shellenv create --shell bash@5.2 --profile strict
eval "$(./dist/shellenv activate)"
./dist/shellenv exec -- echo "hi from $SHELLENV_ENV_NAME"
```
