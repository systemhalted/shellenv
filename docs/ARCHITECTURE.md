# shellenv Architecture

shellenv provides per-project shell sandboxes so scripts can be exercised against specific shells and option profiles without touching a user's login shell.

For the reasoning behind these choices (and known gaps), see `docs/DESIGN.md`.

## Isolation model
shellenv is a **PATH-shimming sandbox**, not an OS-level sandbox. It deliberately uses no chroot, containers, namespaces, or syscalls. Isolation is achieved by four mechanisms, all confined to the child process (or to the shell session the user opts into via `eval`):

1. **`PATH` shimming**: the env's `bin/` is prepended to `PATH`, so project-local tools shadow system tools. `internal/cli/exec.go` (`prependPath`, `resolveCommandPath`, `isExecutable`) resolves the command manually against that modified `PATH`, honoring the hardened Go 1.19+ rule that ignores the current directory (`.`).
2. **Scoped env vars**: `SHELLENV_ACTIVE=1` and `SHELLENV_ENV_NAME=<env>` are set in the child only — they never persist to the parent unless the user `eval`s the `activate` snippet.
3. **Login shell untouched**: no `.bashrc`/`.zshrc`/`.profile` is read or written. Activation is opt-in (`eval "$(shellenv activate)"`) or one-off (`shellenv exec`).
4. **Optional shell-option profiles**: `activate` may source a profile (`strict`/`posix`/`interactive`) to enforce shell options like `set -euo pipefail`.
5. **Containerized execution option (`exec --container <image>`)**: when invoked, the command runs inside an isolated Docker/Podman container. The host workspace is mounted (`-v cwd:cwd`) and the working directory matches. The sandbox `$HOME` and `$TMPDIR` are passed as environment variables to keep file writes inside the host's sandbox folder, while providing the container's isolated network, filesystem, and process namespaces.

### What is and isn't isolated
| Concern | Isolated? | Notes |
| --- | --- | --- |
| Binary resolution (`PATH`) | Yes | Env `bin/` wins; system binaries remain reachable as fallback. |
| `SHELLENV_*` vars & prompt | Yes | Set in the child / opted-in session only. |
| Login shell config | Yes | Never read or modified. |
| `HOME` / dotfiles | `exec` only | `exec` redirects `HOME` to a per-env sandbox (`./.shellenv/<env>/home/`); an `eval`'d `activate` session still uses the real `~/`. |
| `TMPDIR` / `XDG_*` | `exec` only | `exec` points these at the sandbox home; `activate` does not. |
| Inherited env vars (secrets) | **No** | `HOME`, `USER`, tokens, etc. pass through unchanged. |
| Filesystem via absolute paths | **No** | `/etc`, `/var`, `/dev`, absolute paths are the real FS. |
| Network | **No** | No network namespace or filtering. |
| Processes | **No** | No PID namespace; host processes are visible. |
| File descriptors / UID | **No** | Child inherits FDs and runs as the same user. |

> [!NOTE]
> **Container Mode Isolation (`exec --container`):** When executing with the `--container <image>` flag, the sandboxed process runs inside a container namespace. Under this mode, filesystems (outside the workspace), network, processes, and file descriptors are fully isolated by the container runtime.

### Intended use vs. non-use
- **Good for**: testing shell scripts against a declared shell/profile, catching portability issues, and iterating without polluting your real shell setup.
- **Not for**: sandboxing untrusted or malicious code. A hostile script can still reach the network, read secrets from the environment, and touch the real filesystem.

## Layout and data
- **Global home** (`SHELLENV_HOME`, default `~/.shellenv`): created by `shellenv init` with `installs/`, `cache/`, and `tmp/`. A `.initialized` marker is written as part of setup.
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
- **init**: ensures `SHELLENV_HOME` exists, writes `.initialized`.
- **create**: writes `metadata.json` (name, shell, profile, tools placeholder) and ensures `bin/` exists under `./.shellenv/<env>`.
- **activate**: picks an env (arg → `./.shellenv/current` → `default`), verifies it exists, resolves the profile (`SHELLENV_PROFILES` → `./profiles` → alongside the binary) and the declared shell runtime (`$SHELLENV_HOME/installs/<shell>/<version>/bin`, warning on stderr if declared but missing; `--strict-shell` errors instead), and prints shell code that sets `SHELLENV_ACTIVE=1`, `SHELLENV_ENV_NAME`, prepends `bin/` (then the runtime bin) to `PATH`, and prefixes the prompt. Fish activation skips profile sourcing.
- **exec**: uses the same env selection as `activate`, builds a child env with `bin/` prepended (followed by the declared shell's installs bin when resolved — same warning/`--strict-shell` behavior as `activate`) and `SHELLENV_*` vars set, and additionally redirects `HOME`/`TMPDIR`/`XDG_*` to a per-env sandbox (`./.shellenv/<env>/home/`) so scripts don't write to the real home. Under host execution (default), if `--profile` is used, it runs the command *inside* the declared shell after sourcing the env's profile. Under containerized execution (`--container <image>`), it detects the container CLI engine (`docker` or `podman`), mounts the workspace (`-v cwd:cwd`), aligns the working directory, forwards the sandboxed environment variables, and executes the target command wrapped in `sh -c` inside the container to prepend the sandbox `bin/` and source profiles if requested. The child's exit code is mirrored exactly as shellenv's own exit status.
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

## Limitations (current state)
These are known gaps between the tool's intent and its current behavior. Priorities and proposed fixes live in the Roadmap in `docs/DESIGN.md`.

- **`HOME`/`TMPDIR`/`XDG_*` isolation: done for `exec`, open for `activate` (was P0).** `exec` now redirects these to `./.shellenv/<env>/home/`. The `eval`'d `activate` path still uses the real home — isolating an interactive session safely (without breaking the user's shell) is deferred (see roadmap R1).
- **Declared profile via `exec`: done (was P0).** `shellenv exec --profile -- …` sources the env's profile in the declared shell and runs the command inside it. Opt-in (default off) to preserve `exec` semantics for non-shell commands. Caveat: shell *options* (`set -e`, etc.) apply to commands the profiled shell runs directly, not to interpreters the command re-invokes — enforcing options on an arbitrary spawned script is out of scope for PATH-shimming.
- **Declared shell resolution: done (was P1), but installers are still placeholders.** `activate` and `exec` now resolve `metadata.json`'s `Shell` (e.g. `bash@5.2`) to `$SHELLENV_HOME/installs/<shell>/<version>/bin` and prepend it to PATH after the env's `bin/`. A declared-but-uninstalled runtime warns on stderr and falls back to the system shell; `--strict-shell` turns that into an error. Caveats: until real installers land (P2), the installs bin only contains what you put there; resolution is host-only, so `--container` skips it (and rejects `--strict-shell`).
- **Runtime installers are placeholders (P1).** `install`/`uninstall`/`versions` only create/remove directories under `$SHELLENV_HOME/installs/` and write a `placeholder runtime` marker; no shell is downloaded or built.
- **Fish profiles are not sourced (P2).** Fish activation uses `set -gx` syntax and skips profile sourcing (no POSIX `source`), so fish gets weaker option enforcement.
- **No automatic cleanup (P2).** Beyond manual `destroy`, there is no `defer`-based teardown of temporary state.
- **Robustness/test gaps (P2).** Corrupt `metadata.json` handling is untested, and there are no isolation-breach tests.
