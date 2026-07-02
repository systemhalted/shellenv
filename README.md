# shellenv

Per-project shell sandboxes for testing scripts against specific shells and option profiles without polluting your real shell setup.

## Intent
- Treat shells like runtimes: declare the shell version and profile a project expects, and run commands inside that sandbox.
- Keep experiments contained: environments live under `./.shellenv/<name>` and `SHELLENV_HOME` (default `~/.shellenv`), avoiding edits to your login shell.
- Make cross-shell QA easy: quickly swap between `bash`, `zsh`, `fish`, or POSIX-style profiles to catch portability issues early.

> **Scope:** shellenv is a *PATH-shimming* sandbox for not polluting your own shell while you test scripts — not a security sandbox for untrusted code. See `docs/ARCHITECTURE.md` for exactly what is and isn't isolated.

## Disclaimer
This project is provided as-is with no warranties; use at your own risk. See `LICENSE` for details.

## Install & build
Prerequisites: **Go 1.22+** and `make` on your `PATH` (plus [`bats`](https://github.com/bats-core/bats-core) only if you run the integration tests).

```bash
make build          # produces ./dist/shellenv
```

The examples below call the binary as `shellenv`; from a fresh checkout use `./dist/shellenv` (or put `dist/` on your `PATH`).

## Concepts
- **Global home** (`SHELLENV_HOME`, default `~/.shellenv`): created by `shellenv init` with `installs/`, `cache/`, and `tmp/`. Point it at a throwaway directory while experimenting to keep your real home pristine.
- **Project envs** (`./.shellenv/<env>/`): created by `shellenv create`. Each holds `metadata.json` (declared shell + profile), a `bin/` directory for project-local tools, and a sandbox `home/` directory used by `exec` (below).
- **Profiles**: option presets sourced into the shell — built-ins `strict` (`set -euo pipefail`), `posix` (`set -o posix`), and `interactive`. Resolved via `SHELLENV_PROFILES` → `./profiles/<name>.sh` → next to the binary. Fish sessions source a `<name>.fish` variant instead (fish can't source POSIX scripts, and has no `set -e` equivalent — the built-in fish variants only carry comments/exports).

## Quick start
```bash
make build
./dist/shellenv init                                   # set up the global home

# Inside a project directory:
./dist/shellenv create --shell bash@5.2 --profile strict
./dist/shellenv exec -- echo "hi from $SHELLENV_ENV_NAME"   # one-off, isolated
eval "$(./dist/shellenv activate)"                     # or activate your shell session
```

## Two ways to run: `activate` vs `exec` (Host and Container Modes)
Both put the env's `bin/` first on `PATH`. They differ in how far the isolation goes:

| | `eval "$(shellenv activate)"` | `shellenv exec -- <cmd>` | `shellenv exec --container <image> -- <cmd>` |
| --- | --- | --- | --- |
| Prepends env `bin/` to `PATH` | yes | yes | yes (inside container) |
| Sets `SHELLENV_*` variables | yes | yes | yes (inside container) |
| Sandboxes `HOME` / `TMPDIR` | **no** (real `~`) | **yes** (`./.shellenv/<env>/home/`) | **yes** (mounted on container) |
| Network / Process isolation | **no** | **no** | **yes** (container namespaces) |
| System filesystem isolation | **no** | **no** | **yes** (container namespaces) |
| Changes current shell session | yes (until you reset it) | no (subprocess only) | no (container execution) |
| Propagates command exit code | n/a | yes | yes |

- Use **`activate`** for an interactive session where you want the env's tools and profile in your prompt.
- Use **`exec`** (Host Mode) for one-off or scripted runs you want local-user contained — including writes to `$HOME` and temp files.
- Use **`exec --container <image>`** (Container Mode) for system-mutating scripts (like installation scripts) that need full filesystem, process, and network namespace isolation.

For all, the env is chosen as: the name you pass → `./.shellenv/current` (set by `shellenv use`) → `default`.

## Walkthrough: test a script in isolation
This runs a script through `exec` and shows that its `$HOME` writes land in the sandbox (your real home is untouched) and that its exit code is propagated.

```bash
# Keep the global home off your real ~ while experimenting.
export SHELLENV_HOME="$(mktemp -d)"     # or: mkdir /tmp/se-home && export SHELLENV_HOME=/tmp/se-home

# From inside your project directory:
shellenv create --shell bash@5.2 --profile strict   # makes ./.shellenv/default

# A script that writes to $HOME and fails:
cat > probe.sh <<'EOF'
#!/bin/sh
: > "$HOME/wrote-here"     # creates a file in $HOME
echo "HOME is $HOME"
exit 5
EOF
chmod +x probe.sh

shellenv exec -- ./probe.sh
echo "shellenv exit: $?"               # -> 5  (the child's real status)

ls .shellenv/default/home/wrote-here   # the write landed in the sandbox
test -e "$HOME/wrote-here" && echo "leaked!" || echo "real HOME untouched"
```

`exec` resolves commands against the env's `bin/` first, so dropping a tool in `./.shellenv/default/bin/` lets it shadow the system copy for the duration of the run.

## Applying a profile with `exec --profile`
`--profile` sources the env's declared profile in the declared shell before running your command, so the profile's exported settings and shell options are in effect:

```bash
# strict mode (set -e) aborts a failing direct command:
shellenv exec --profile -- false && echo reached || echo "aborted ($?)"
# -> aborted (1)
```

**Caveat:** the profile's shell *options* (`set -e`, `set -o pipefail`, …) apply to commands the profiled shell runs directly. A command that re-invokes an interpreter — a script with its own shebang, or `bash -c '…'` — starts a fresh shell and inherits only the profile's **exported environment**, not its options. (Well-written scripts set their own `set -euo pipefail`.) Profiles resolve from `SHELLENV_PROFILES`, then `./profiles/<name>.sh`, then beside the binary — so running `./dist/shellenv` finds the built-ins shipped in this repo; an installed binary needs `./profiles` or `SHELLENV_PROFILES`.

## Pinning the declared shell version
`create --shell bash@5.2` records the shell your project expects, and `activate`/`exec` now act on it: if `$SHELLENV_HOME/installs/bash/5.2/bin` exists (created by `shellenv install bash@5.2`), it is added to `PATH` right after the env's `bin/`, so the pinned interpreter wins over the system one while project-local tools still win over both.

If the declared version is **not** installed, the command still runs against the system shell but prints a warning to stderr. Pass `--strict-shell` (on `activate` or `exec`) to fail instead:

```bash
shellenv exec --strict-shell -- ./run-tests.sh
# error: declared shell "bash@5.2" is not installed (run 'shellenv install bash@5.2', or omit --strict-shell)
```

`shellenv install` builds the runtime from the official source tarball: it downloads into `$SHELLENV_HOME/cache/`, verifies the SHA-256 for pinned versions (`bash@5.2`, `zsh@5.9` — other versions install with an unverified-download warning), and runs `configure`/`make`/`make install` (a few minutes, once per version; build output lands in a `build.log` if anything fails). This needs `cc`/`gcc`, `make`, and `tar` — run `shellenv doctor` to check. Supported today: **bash** and **zsh**.

**Caveat:** resolution is host-only — `exec --container` skips it (pick an image that provides the shell) and rejects `--strict-shell`.

## Exit codes & CI
`shellenv exec` exits with the command's exact status (e.g. `exit 5` above), so it composes cleanly with `set -e` and CI pipelines. Runtime failures print a single clean line rather than a usage dump:

```bash
shellenv exec --profile -- ./run-tests.sh || exit $?   # fail the build on a non-zero test run
```

## Command reference
- `shellenv init`: create the global home.
- `shellenv create [--name default] --shell <shell>@<ver> [--profile strict|posix|interactive] [--with-tools]`: scaffold a project env.
- `shellenv use <env>` / `shellenv list` / `shellenv destroy <env>`: set the current env, list envs, or remove one.
- `shellenv activate [<env>] [--shell-type bash|zsh|fish] [--strict-shell]`: print an activation snippet to `eval`.
- `shellenv exec [<env>] [--profile] [--strict-shell] [--ephemeral] [--container <image>] -- <cmd> [args]`: run a command in the env without activating your shell (sandboxes `HOME`/`TMPDIR`/`XDG_*`, propagates the exit code). `--ephemeral` swaps the persistent sandbox home for a throwaway one deleted after the run. If `--container` is provided, executes inside the specified Docker or Podman image with mounted workspace.
- `shellenv which <binary>`: resolve a tool, preferring the active env's `bin/`.
- `shellenv install <shell>@<ver>` / `shellenv uninstall …` / `shellenv versions`: build a runtime from the official source tarball (bash and zsh; needs `cc`/`make`/`tar`), remove one, or list what's installed.
- `shellenv doctor`: quick health check of the global home.

## Environment variables
- `SHELLENV_HOME`: global state root (default `~/.shellenv`).
- `SHELLENV_PROFILES`: directory checked first when resolving a profile.
- `SHELLENV_ACTIVE`: set to `1` inside an activated session or an `exec` child.
- `SHELLENV_ENV_NAME`: the active env's name (handy for prompts and debugging).

## Docs
- Contributor workflow and standards: `CONTRIBUTING.md`.
- Architecture, isolation model, and flows: `docs/ARCHITECTURE.md`.
- Design decisions, rationale, and roadmap: `docs/DESIGN.md`.
- Task notes and change log: `docs/Task.md`.

## Testing & dev notes
- Unit tests: `make test`.
- Integration tests (require `bats`): `SHELLENV_HOME=$(mktemp -d) bats -r test/integration`. The real-source-build test is skipped unless `SHELLENV_TEST_REAL_INSTALL=1` is set (network + several minutes).
- If your environment restricts the default Go cache, use a repo-local one: `GOCACHE=$PWD/.cache/go-build go test ./...`.
- Keep experiments isolated by pointing `SHELLENV_HOME` at a temp directory when hacking on the tool.
