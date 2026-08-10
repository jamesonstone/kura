.DEFAULT_GOAL := help

BINARY_NAME := kura
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || printf '%s' dev)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: help build install fmt fmt-check vet test test-race lint source-size check release-check snapshot clean

help:
	@printf '%s\n' 'Kura developer workflow'
	@printf '%s\n' ''
	@printf '%s\n' '  make build          Build bin/kura'
	@printf '%s\n' '  make install        Install kura with go install'
	@printf '%s\n' '  make check          Run formatting, vet, tests, and size checks'
	@printf '%s\n' '  make test-race      Run the complete race-enabled test suite'
	@printf '%s\n' '  make lint           Run golangci-lint'
	@printf '%s\n' '  make release-check  Validate GoReleaser configuration'
	@printf '%s\n' '  make snapshot       Build a local GoReleaser snapshot'

build:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/kura

install:
	go install $(LDFLAGS) ./cmd/kura

fmt:
	gofmt -w cmd internal

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || { gofmt -l cmd internal; exit 1; }

vet:
	go vet ./...

test:
	go test ./... -count=1

test-race:
	go test -race ./... -count=1

lint:
	golangci-lint run ./...

source-size:
	@find cmd internal -type f -name '*.go' -print0 | \
		xargs -0 awk 'FNR == 1 { file = FILENAME } FNR > 300 { print file ":" FNR; failed = 1; nextfile } END { exit failed }'

check: fmt-check vet test source-size

release-check:
	goreleaser check

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
