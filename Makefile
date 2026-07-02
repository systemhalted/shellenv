SHELL := /bin/bash

.PHONY: build test itest

build:
	mkdir -p dist
	go build -o dist/shellenv ./cmd/shellenv

test:
	go test ./...

# Integration tests need a built binary and bats; a temp SHELLENV_HOME keeps
# them off the real ~/.shellenv.
itest: build
	SHELLENV_HOME=$$(mktemp -d) bats -r test/integration
