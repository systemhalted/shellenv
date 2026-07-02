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
