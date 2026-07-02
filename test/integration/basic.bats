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

  run bash -lc ''"$BIN"' install bash@5.2'
  [ "$status" -eq 0 ]

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

@test "exec with container" {
  [ -x "$BIN" ] || skip "build not found at $BIN (run: make build)"
  which docker >/dev/null 2>&1 || skip "docker not found"

  run bash -lc 'mkdir -p proj-container && cd proj-container && '"$BIN"' create --shell bash@5.2'
  [ "$status" -eq 0 ]

  run bash -lc 'cd proj-container && '"$BIN"' exec --container alpine -- echo "hello container"'
  [ "$status" -eq 0 ]
  [[ "$output" =~ "hello container" ]]

  rm -rf proj-container
}
