SHELL := /bin/bash

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Pinned here (not in CI) so local runs and CI always agree. Both tools need a
# recent Go toolchain to build; the module itself still targets go 1.22.
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.7.0
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.6.0

.PHONY: build test itest lint vulncheck man

build:
	mkdir -p dist
	go build -ldflags "-X github.com/systemhalted/shellenv/internal/cli.version=$(VERSION)" -o dist/shellenv ./cmd/shellenv

test:
	go test ./...

# Integration tests need a built binary and bats; a temp SHELLENV_HOME keeps
# them off the real ~/.shellenv.
itest: build
	SHELLENV_HOME=$$(mktemp -d) bats -r test/integration

lint:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi
	go vet ./...
	go run $(STATICCHECK) ./...

# Scans with the running toolchain's stdlib: a finding fixed in a newer Go
# patch release means "update your toolchain", per the blocking-gate policy.
vulncheck:
	go run $(GOVULNCHECK) ./...

# Generated from the live Cobra tree so pages always match --help; man/ is
# gitignored and rebuilt on demand (release tarballs bundle it).
man:
	go run ./cmd/gen-man man
