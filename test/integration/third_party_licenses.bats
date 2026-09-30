#!/usr/bin/env bats

setup() {
  export SCRIPT="$BATS_TEST_DIRNAME/../../scripts/third-party-licenses.sh"
  command -v go >/dev/null || skip "go not on PATH"
}

@test "third-party licenses cover the modules linked on linux" {
  run env GOOS=linux GOARCH=amd64 "$SCRIPT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"github.com/spf13/cobra "* ]]
  [[ "$output" == *"github.com/spf13/pflag "* ]]
  [[ "$output" == *"Apache License"* ]]
  # mousetrap is only linked into Windows builds.
  [[ "$output" != *"github.com/inconshreveable/mousetrap"* ]]
  # Build-time-only man-page modules are not in the binary.
  [[ "$output" != *"go-md2man"* ]]
}

@test "third-party licenses follow the target platform" {
  run env GOOS=windows GOARCH=amd64 "$SCRIPT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"github.com/inconshreveable/mousetrap "* ]]
}
