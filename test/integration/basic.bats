#!/usr/bin/env bats

setup() {
  export SHELLENV_HOME="$BATS_TMPDIR/home"
  mkdir -p "$SHELLENV_HOME"
  export BIN="$BATS_TEST_DIRNAME/../../dist/shellenv"
}

@test "build exists" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"
}

@test "create + activate + exec" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  # Pre-create the runtime dir so activate/exec see bash@5.2 as installed
  # without a real (multi-minute) source build.
  mkdir -p "$SHELLENV_HOME/installs/bash/5.2/bin"

  run bash -lc 'mkdir -p proj && cd proj && '"$BIN"' create --shell bash@5.2 --profile strict'
  [ "$status" -eq 0 ]

  run bash -lc 'cd proj && ACT=$('"$BIN"' activate); eval "$ACT"; echo $SHELLENV_ENV_NAME'
  [ "$status" -eq 0 ]
  [ "$output" = "default" ]

  run bash -lc 'cd proj && echo -e "#!/usr/bin/env bash\necho hi" > ./.shellenv/default/bin/hi && chmod +x ./.shellenv/default/bin/hi && '"$BIN"' exec -- hi'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "hi" ]]
  rm -rf proj
}

@test "exec warns and falls back when declared shell is not installed" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  run bash -lc 'mkdir -p proj-warn && cd proj-warn && '"$BIN"' create --shell bash@9.9'
  [ "$status" -eq 0 ]

  run bash -lc 'cd proj-warn && '"$BIN"' exec -- echo fallback-ok'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "fallback-ok" ]]
  [[ "$output" =~ "not installed" ]]

  rm -rf proj-warn
}

@test "exec --strict-shell fails when declared shell is not installed" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  run bash -lc 'mkdir -p proj-strict && cd proj-strict && '"$BIN"' create --shell bash@9.9'
  [ "$status" -eq 0 ]

  run bash -lc 'cd proj-strict && '"$BIN"' exec --strict-shell -- echo should-not-run'
  [ "$status" -ne 0 ]
  [[ "$output" =~ "not installed" ]]
  [[ ! "$output" =~ "should-not-run" ]]

  rm -rf proj-strict
}

@test "exec --ephemeral leaves no sandbox home behind" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  run bash -lc 'mkdir -p proj-eph && cd proj-eph && '"$BIN"' create --shell bash@9.9'
  [ "$status" -eq 0 ]

  run bash -lc 'cd proj-eph && printf "#!/bin/sh\ntouch \"\$HOME/marker\"\necho \"HOME=\$HOME\"\n" > ./.shellenv/default/bin/homeprobe && chmod +x ./.shellenv/default/bin/homeprobe && '"$BIN"' exec --ephemeral -- homeprobe 2>/dev/null'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "home-ephemeral" ]]

  run bash -lc 'ls proj-eph/.shellenv/default/ | grep home-ephemeral || echo clean'
  [ "$output" = "clean" ]

  rm -rf proj-eph
}

@test "activate --isolate-home + deactivate round-trip" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"

  run bash -lc 'mkdir -p proj-iso && cd proj-iso && '"$BIN"' create --shell bash@9.9'
  [ "$status" -eq 0 ]

  run bash -c 'cd proj-iso
    orig_home="$HOME"
    eval "$('"$BIN"' activate --isolate-home 2>/dev/null)"
    touch "$HOME/iso-marker"
    [ -f ./.shellenv/default/home/iso-marker ] || { echo "marker not in sandbox"; exit 1; }
    eval "$('"$BIN"' deactivate)"
    [ "$HOME" = "$orig_home" ] || { echo "HOME not restored"; exit 1; }
    [ -z "${SHELLENV_ACTIVE-}" ] || { echo "SHELLENV_ACTIVE leaked"; exit 1; }
    echo iso-ok'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "iso-ok" ]]

  rm -rf proj-iso
}

@test "install builds a real bash from source (opt-in)" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"
  [ -n "${SHELLENV_TEST_REAL_INSTALL:-}" ] || skip "set SHELLENV_TEST_REAL_INSTALL=1 to run the real source build (network + several minutes)"

  run bash -lc ''"$BIN"' install bash@5.2'
  [ "$status" -eq 0 ]

  run "$SHELLENV_HOME/installs/bash/5.2/bin/bash" -c 'echo "$BASH_VERSION"'
  [ "$status" -eq 0 ]
  [[ "$output" =~ ^5\.2 ]]
}

@test "exec with container" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"
  # Same engines findContainerEngine accepts. SHELLENV_TEST_REQUIRE_CONTAINER=1
  # (set in CI on Linux) turns a silent skip into a failure so the container
  # path can't quietly stop being tested.
  if ! command -v docker >/dev/null 2>&1 && ! command -v podman >/dev/null 2>&1; then
    if [ -n "${SHELLENV_TEST_REQUIRE_CONTAINER:-}" ]; then
      echo "container engine required (SHELLENV_TEST_REQUIRE_CONTAINER=1) but none found" >&2
      return 1
    fi
    skip "no container engine (docker/podman) found"
  fi

  run bash -lc 'mkdir -p proj-container && cd proj-container && '"$BIN"' create --shell bash@5.2'
  [ "$status" -eq 0 ]

  run bash -lc 'cd proj-container && '"$BIN"' exec --container alpine -- echo "hello container"'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "hello container" ]]

  rm -rf proj-container
}
