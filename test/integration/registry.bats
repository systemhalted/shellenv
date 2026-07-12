#!/usr/bin/env bats

# End-to-end checks for the advisory env registry (decision 15): create/destroy
# maintain $SHELLENV_HOME/registry.json, uninstall warns across projects, and
# stale entries are pruned rather than warned about.

setup() {
  export SHELLENV_HOME="$BATS_TMPDIR/registry-home"
  rm -rf "$SHELLENV_HOME" "$BATS_TMPDIR/reg-proj-a" "$BATS_TMPDIR/reg-proj-b"
  mkdir -p "$SHELLENV_HOME"
  export BIN="$BATS_TEST_DIRNAME/../../dist/shellenv"
}

@test "list --all sees envs registered from other directories" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  run bash -lc 'mkdir -p "$BATS_TMPDIR/reg-proj-a" && cd "$BATS_TMPDIR/reg-proj-a" && '"$BIN"' create --shell bash@5.2'
  [ "$status" -eq 0 ]
  run bash -lc 'mkdir -p "$BATS_TMPDIR/reg-proj-b" && cd "$BATS_TMPDIR/reg-proj-b" && '"$BIN"' create --shell zsh@5.9'
  [ "$status" -eq 0 ]

  run bash -lc 'cd "$BATS_TMPDIR/reg-proj-b" && '"$BIN"' list --all'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "reg-proj-a" ]]
  [[ "$output" =~ "reg-proj-b" ]]
  [[ "$output" =~ "bash@5.2" ]]
  [[ "$output" =~ "zsh@5.9" ]]
}

@test "uninstall warns about declaring envs in other projects and prunes vanished ones" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  # A fake installed runtime so uninstall has something to remove.
  mkdir -p "$SHELLENV_HOME/installs/bash/5.2/bin"

  run bash -lc 'mkdir -p "$BATS_TMPDIR/reg-proj-a" && cd "$BATS_TMPDIR/reg-proj-a" && '"$BIN"' create --shell bash@5.2'
  [ "$status" -eq 0 ]

  # From an unrelated directory, only the registry can reveal proj-a.
  run bash -lc 'mkdir -p "$BATS_TMPDIR/reg-proj-b" && cd "$BATS_TMPDIR/reg-proj-b" && '"$BIN"' uninstall bash@5.2'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "reg-proj-a" ]]
  [[ "$output" =~ "still declares bash@5.2" ]]

  # The project vanishes without a destroy: no warning, entry pruned.
  rm -rf "$BATS_TMPDIR/reg-proj-a"
  mkdir -p "$SHELLENV_HOME/installs/bash/5.2/bin"
  run bash -lc 'cd "$BATS_TMPDIR/reg-proj-b" && '"$BIN"' uninstall bash@5.2'
  [ "$status" -eq 0 ]
  [[ ! "$output" =~ "reg-proj-a" ]]
  run grep -c "reg-proj-a" "$SHELLENV_HOME/registry.json"
  [ "$status" -ne 0 ]
}

@test "destroy removes the env from the registry" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  run bash -lc 'mkdir -p "$BATS_TMPDIR/reg-proj-a" && cd "$BATS_TMPDIR/reg-proj-a" && '"$BIN"' create --shell bash@5.2'
  [ "$status" -eq 0 ]
  run grep -c "reg-proj-a" "$SHELLENV_HOME/registry.json"
  [ "$status" -eq 0 ]

  run bash -lc 'cd "$BATS_TMPDIR/reg-proj-a" && '"$BIN"' destroy default'
  [ "$status" -eq 0 ]
  run grep -c "reg-proj-a" "$SHELLENV_HOME/registry.json"
  [ "$status" -ne 0 ]
}
