BIN := kohrdp
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
BASE_VERSION := $(shell tr -d '\n' < VERSION)
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || printf $(BASE_VERSION))
LDFLAGS := -X main.version=$(VERSION)
TARBALL ?=

.PHONY: help build install uninstall install-tarball test vet fmt check dist-arch release-tarball

.DEFAULT_GOAL := help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -ldflags "$(LDFLAGS)" -o $(BIN) .

install: build ## Build and install to $(BINDIR)
	install -Dm755 $(BIN) "$(BINDIR)/$(BIN)"

uninstall: ## Remove the installed binary
	rm -f "$(BINDIR)/$(BIN)"

install-tarball: ## Install from a built tarball (TARBALL=...)
	test -n "$(TARBALL)"
	rm -rf /tmp/kohrdp-tmp/$(BIN)-install
	mkdir -p /tmp/kohrdp-tmp/$(BIN)-install
	tar -xzf "$(TARBALL)" -C /tmp/kohrdp-tmp/$(BIN)-install
	install -Dm755 /tmp/kohrdp-tmp/$(BIN)-install/$(BIN)-*/$(BIN) "$(BINDIR)/$(BIN)"

test: ## Run tests
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format source with gofmt
	gofmt -w main.go internal/config/*.go internal/keyring/*.go internal/rdp/*.go internal/ui/*.go

check: test vet build ## Run tests, vet, and build

dist-arch: ## Build the Arch package source under dist/arch/
	rm -rf dist/arch
	mkdir -p dist/arch
	tar -czf "dist/arch/$(BIN)-$(VERSION).tar.gz" --exclude ./dist --transform 's,^\.,$(BIN)-$(VERSION),' .
	sed "s,@VERSION@,$(VERSION),g" packaging/arch/PKGBUILD > dist/arch/PKGBUILD

release-tarball: build ## Build a versioned binary tarball under dist/release/
	rm -rf dist/release
	mkdir -p "dist/release/$(BIN)-$(VERSION)"
	install -m755 "$(BIN)" "dist/release/$(BIN)-$(VERSION)/$(BIN)"
	install -m644 README.md "dist/release/$(BIN)-$(VERSION)/README.md"
	tar -czf "dist/release/$(BIN)-$(VERSION)-$(shell go env GOOS)-$(shell go env GOARCH).tar.gz" -C dist/release "$(BIN)-$(VERSION)"
