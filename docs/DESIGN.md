# shellenv Design Decisions

This document records *why* shellenv is built the way it is — the rationale and trade-offs
behind the architecture. For *what* the pieces are and how they connect, see
`docs/ARCHITECTURE.md`. For user-facing usage, see `README.md`.

Each decision is stated as **Decision / Why / Trade-off / Status**.

## 1. PATH-shimming over OS-level isolation
- **Decision**: Achieve isolation by prepending the env `bin/` to `PATH` and scoping
  `SHELLENV_*` vars to the child, rather than using chroot, containers, or namespaces.
- **Why**: Portable across macOS/Linux, needs no root, no daemon, and no platform-specific
  syscalls. It directly serves the primary goal — *not polluting the user's real shell* —
  without the weight of a true sandbox.
- **Trade-off**: The isolation boundary is weak. `HOME`, `/tmp`, network, processes, and
  inherited env vars all leak to the host (see the boundary table in `ARCHITECTURE.md`).
  Not safe for untrusted code.
- **Status**: Current and foundational. P0 roadmap items tighten the boundary (HOME/TMPDIR)
  without abandoning this model.

## 2. Per-project `./.shellenv/<env>/` + global `SHELLENV_HOME` split
- **Decision**: Keep project-scoped config (`metadata.json`, `bin/`, `current`) inside the
  repo under `./.shellenv/`, and keep shared/installed runtimes under a global
  `SHELLENV_HOME` (default `~/.shellenv`, with `installs/`, `cache/`, `tmp/`).
- **Why**: Mirrors the pyenv/rbenv mental model. Project config travels with the repo and is
  reviewable; heavy installed runtimes are shared across projects instead of duplicated.
- **Trade-off**: Two roots to reason about; `SHELLENV_HOME` is global mutable state that the
  docs recommend pointing at a throwaway dir during development.
- **Status**: Current.

## 3. `activate` prints code to `eval` (not a managed subshell)
- **Decision**: `shellenv activate` writes shell code to stdout for the user to
  `eval "$(shellenv activate)"`, instead of spawning a managed subshell.
- **Why**: The user's shell stays authoritative; activation is explicit and opt-in, and it
  composes with whatever the user is already doing. It also keeps the tool stateless about
  shell lifecycle.
- **Trade-off**: Changes (PATH, prompt, sourced profile options) persist in that session
  until the user resets them; there is no automatic deactivation.
- **Status**: Current.

## 4. `exec` resolves the command path manually
- **Decision**: `exec` walks the modified `PATH` itself (`resolveCommandPath` /
  `isExecutable` in `internal/cli/exec.go`) rather than relying on `exec.LookPath`/the OS.
- **Why**: Guarantees the project-local `bin/` wins, and honors the hardened Go 1.19+ rule
  that never searches the current directory (`.`) implicitly — a deliberate security choice.
- **Trade-off**: A little reimplemented lookup logic to maintain.
- **Status**: Current.

## 5. Cobra + `RunE` everywhere, with a testable `ExecuteWithArgs`
- **Decision**: Use Cobra for dispatch; every command implements `RunE` (error-returning),
  and `internal/cli/root.go` exposes `ExecuteWithArgs(args, stdout, stderr)` alongside
  `Execute()`.
- **Why**: `RunE` lets failures propagate to a non-zero exit code cleanly.
  `ExecuteWithArgs` injects args and writers so commands are unit-testable without touching
  real stdout/stderr or `os.Args`.
- **Trade-off**: Slightly more plumbing than calling `cmd.Execute()` directly.
- **Status**: Current.

## 6. Profiles are sourced shell scripts, not Go config
- **Decision**: Ship `strict`/`posix`/`interactive` as plain `.sh` files under `profiles/`,
  resolved via `SHELLENV_PROFILES` → `./profiles/<name>.sh` → alongside the binary.
- **Why**: Users can add or override profiles with zero recompilation, and the contents are
  exactly the shell options they'd write by hand (`set -euo pipefail`, `set -o posix`).
- **Trade-off**: Profiles are shell-specific. Fish can't `source` POSIX scripts, so fish
  currently gets no profile (a documented limitation).
- **Status**: Current; fish support is on the roadmap.

## 7. `internal/` package split (`env` / `project` / `shell` / `cli`)
- **Decision**: `cli` orchestrates commands; `env` owns `SHELLENV_HOME` resolution;
  `project` owns per-project metadata/listing/current-env; `shell` owns activation-snippet
  generation and profile resolution.
- **Why**: Separation of concerns — the non-`cli` packages stay relatively pure and
  independently testable, and the command layer stays thin.
- **Trade-off**: Some indirection for a small codebase.
- **Status**: Current.

## 8. Runtime installers are deliberately placeholders (for now)
- **Decision**: `install`/`uninstall`/`versions` create/remove directory structure under
  `$SHELLENV_HOME/installs/` and write a `placeholder runtime` marker, without actually
  provisioning a shell.
- **Why**: Real shell provisioning (download/build of pinned versions) is the hard part and
  was staged for later so the rest of the workflow (create/activate/exec/profiles) could
  ship and be exercised first.
- **Trade-off**: The headline "declare and pin a shell version" promise isn't fully real yet —
  the declared version is recorded but never resolved.
- **Status**: Roadmap (item R5 below).

## 9. Silence Cobra's usage/error output; report errors centrally
- **Decision**: Set `SilenceUsage` and `SilenceErrors` on the root command, print the error
  message once from `Execute()`, and propagate a command's real exit code via an `exitError`
  type (`internal/cli/root.go`, `internal/cli/exec.go`).
- **Why**: `shellenv exec` is a command *runner* — a command that exits non-zero is a normal,
  expected outcome, not a misuse of shellenv. Cobra's default behavior dumped the full usage
  block and an `Error:` line on every failure (and `Execute()` then printed the error a second
  time), so `shellenv exec -- sh -c 'exit 7'` produced a wall of noise and always exited `1`.
  With this change the child's status is mirrored (`exit 7`) and runtime errors read as a
  single clean line.
- **Trade-off**: Silencing is **global**, so genuine argument/usage errors no longer
  auto-append Cobra's usage block — they surface only the error message (our own commands,
  e.g. `exec`, still print an explicit `usage: …` string where it helps). The full usage text
  remains available via `shellenv <cmd> --help`. We judged a clean runner experience worth
  more than auto-usage on bad args; revisit if users find arg errors unclear (e.g. by
  re-enabling usage only for `cobra`-level argument-validation failures).
- **Status**: Current.
 
## 10. Explicit Container Execution Driver (`--container <image>`)
- **Decision**: Introduce an explicit container execution driver to allow running script commands inside a Docker or Podman container.
- **Why**: Many scripts (like system setup wrappers) mutate global host resources (package manager installs, absolute directories outside home). These cannot be safely isolated via local PATH shimming alone. The container driver runs the command inside a clean container namespace with mounted workspaces and redirected sandbox folders.
- **Trade-off**: Requires a container CLI (Docker or Podman) to be running on the host machine.
- **Status**: Current.

## 11. Declared shell resolution: warn-and-fallback, strictness opt-in (`--strict-shell`)
- **Decision**: `activate`/`exec` resolve `metadata.Shell` (`<shell>@<version>`) to
  `$SHELLENV_HOME/installs/<shell>/<version>/bin` via `env.ResolveRuntime` and prepend it to
  PATH *after* the env's `bin/`. A declared-but-uninstalled runtime produces a stderr warning
  and falls back to the system shell; `--strict-shell` makes it a hard error instead.
- **Why**: Installers are placeholders until R5, so hard-erroring on a missing runtime would
  break every `create → exec` workflow that exists today. The warning ends the previous
  silent fallback; the flag gives users who want enforced pinning the roadmap's strict
  behavior. Opt-in-flag precedent: `exec --profile` (R2). Env bin stays first on PATH so
  project-local tools keep their existing precedence.
- **Trade-off**: A warning is easier to miss than an error. `--strict-shell` with an
  unversioned declaration (`shell` without `@version`) also errors — strictness that silently
  enforces nothing would be a footgun. Resolution is host-only: `--container` skips it (a
  host installs path is meaningless inside the image) and rejects `--strict-shell`.
- **Status**: Current.

## 12. Remove `shims/` in favor of per-env PATH pinning
- **Decision**: Drop the unused `$SHELLENV_HOME/shims` directory and everything that
  referenced it: `env.ShimsDir()`, the `init` PATH hint ("add …/shims to your rc file"),
  and `doctor`'s shims line. Existing on-disk `shims/` dirs are left alone (harmless, empty).
- **Why**: Nothing ever wrote into `shims/`. A global shims dir on the login PATH is the
  pyenv model for *global* version switching, which contradicts this tool's promise of
  per-project envs that never touch the login shell — `init`'s only PATH hint asked users to
  edit their rc file for a no-op. R3 delivers version pinning per env at activate/exec time
  instead.
- **Trade-off**: If R5 (real installers) ever wants pyenv-style shims, this is a ~20-line
  git revert away; the decision record stays so the context isn't lost.
- **Status**: Current.

---

# Roadmap (gap-closing)

Prioritized fixes for the limitations listed in `docs/ARCHITECTURE.md`. P0 items are small,
high-value, and directly serve "don't impact the host."

- **R1 (P0) — Isolate `HOME`/`TMPDIR`/`XDG_*` for `exec`. _Done._** `exec` creates a per-env
  sandbox home (`./.shellenv/<env>/home/`, via `project.SandboxHomeDir`) and overrides
  `HOME`/`TMPDIR`/`XDG_CONFIG_HOME`/`XDG_CACHE_HOME`/`XDG_DATA_HOME` in the child env only.
  Remaining: doing the same for an `eval`'d `activate` session without breaking the user's
  interactive shell (likely opt-in).
- **R2 (P0) — Make `exec` honor the declared profile. _Done._** `exec --profile` sources the
  env's profile (resolved via `shell.ResolveProfile`) in the declared shell
  (`profileShell` derives the interpreter from `metadata.Shell`) and runs the command inside
  it. Opt-in by design so non-shell commands keep their current behavior. Known limit: shell
  options apply only to commands the profiled shell runs directly — a re-invoked interpreter
  (script shebang, `bash -c`) inherits only the profile's exported environment, which is
  inherent to PATH-shimming rather than OS-level isolation.
- **R3 (P1) — Resolve and use the declared shell version. _Done._** `activate` and `exec`
  resolve `metadata.Shell` under `$SHELLENV_HOME/installs/<shell>/<version>/bin` (via
  `env.ResolveRuntime`) and prepend it to PATH after the env's `bin/` (project-local tools
  keep winning). A declared-but-missing runtime warns on stderr and falls back to the system
  shell; `--strict-shell` (on both commands) turns that into a hard error (see decision 11).
  Host-only: `--container` skips resolution, and combining it with `--strict-shell` errors.
  Remaining: until R5 lands, installed runtimes are placeholder dirs, so pinning is only as
  real as what you put in the installs bin.
- **R4 (P1) — Use or remove `shims/`. _Done (removed)._** Dropped the unused directory,
  `env.ShimsDir()`, `init`'s rc-file PATH hint, and `doctor`'s shims line (see decision 12).
  R3's per-env PATH pinning supersedes the global-shims model; trivially reversible via git
  if R5 ever wants shims back.
- **R5 (P2, large) — Real runtime installers.** Replace placeholder `install.go` with actual
  download/build of pinned shell versions.
- **R6 (P2) — Fish profiles, cleanup, robustness.** Add fish-syntax profile variants and
  source them; add an optional `exec --ephemeral` with `defer`-based teardown; handle corrupt
  `metadata.json` gracefully; add isolation-breach tests.
