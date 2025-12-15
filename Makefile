SHELL := /bin/bash

.PHONY: build test

build:
	mkdir -p dist
	go build -o dist/shellenv ./cmd/shellenv

test:
	go test ./...
