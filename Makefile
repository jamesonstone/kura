.DEFAULT_GOAL := help

BINARY_NAME := kura
WORKTREE_BINARY_NAME := git-wt
KURA_PREFIX ?= $(HOME)/.local
KURA_BIN_DIR ?= $(KURA_PREFIX)/bin
KURA_MAN_DIR ?= $(KURA_PREFIX)/share/man/man1
KURA_STATE_DIR ?= $(HOME)/.local/state/kura
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || printf '%s' dev)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: help build build-kura install install-kura install-git-wt fmt fmt-check vet test test-race lint source-size check release-check snapshot clean

help:
	@printf '%s\n' 'Kura developer workflow'
	@printf '%s\n' ''
	@printf '%s\n' '  make build          Build bin/kura and refresh the user-local git-wt alias'
	@printf '%s\n' '  make build-kura     Build bin/kura without host installation'
	@printf '%s\n' '  make install        Install kura, git-wt, and the git-wt manual page'
	@printf '%s\n' '  make install-git-wt Install the current build as the user-local git-wt alias'
	@printf '%s\n' '  make check          Run formatting, vet, tests, and size checks'
	@printf '%s\n' '  make test-race      Run the complete race-enabled test suite'
	@printf '%s\n' '  make lint           Run golangci-lint'
	@printf '%s\n' '  make release-check  Validate GoReleaser configuration'
	@printf '%s\n' '  make snapshot       Build a local GoReleaser snapshot'

build: build-kura install-git-wt

build-kura:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/kura

install: build-kura install-kura install-git-wt

install-kura: build-kura
	mkdir -p $(KURA_BIN_DIR)
	install -m 0755 bin/$(BINARY_NAME) $(KURA_BIN_DIR)/$(BINARY_NAME)

install-git-wt: build-kura
	bin/$(BINARY_NAME) install --force --bin-dir $(KURA_BIN_DIR) --man-dir $(KURA_MAN_DIR) --state-dir $(KURA_STATE_DIR) $(WORKTREE_BINARY_NAME)

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
