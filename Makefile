BIN := rdpkoh
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
BASE_VERSION := $(shell tr -d '\n' < VERSION)
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || printf $(BASE_VERSION))
LDFLAGS := -X main.version=$(VERSION)
TARBALL ?=

.PHONY: build install uninstall install-tarball test vet fmt check dist-arch release-tarball

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) .

install: build
	install -Dm755 $(BIN) "$(BINDIR)/$(BIN)"

uninstall:
	rm -f "$(BINDIR)/$(BIN)"

install-tarball:
	test -n "$(TARBALL)"
	rm -rf /tmp/opencode/$(BIN)-install
	mkdir -p /tmp/opencode/$(BIN)-install
	tar -xzf "$(TARBALL)" -C /tmp/opencode/$(BIN)-install
	install -Dm755 /tmp/opencode/$(BIN)-install/$(BIN)-*/$(BIN) "$(BINDIR)/$(BIN)"

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w main.go internal/config/*.go internal/keyring/*.go internal/rdp/*.go internal/ui/*.go

check: test vet build

dist-arch:
	rm -rf dist/arch
	mkdir -p dist/arch
	tar -czf "dist/arch/$(BIN)-$(VERSION).tar.gz" --exclude ./dist --transform 's,^\.,$(BIN)-$(VERSION),' .
	sed "s,@VERSION@,$(VERSION),g" packaging/arch/PKGBUILD > dist/arch/PKGBUILD

release-tarball: build
	rm -rf dist/release
	mkdir -p "dist/release/$(BIN)-$(VERSION)"
	install -m755 "$(BIN)" "dist/release/$(BIN)-$(VERSION)/$(BIN)"
	install -m644 README.md "dist/release/$(BIN)-$(VERSION)/README.md"
	tar -czf "dist/release/$(BIN)-$(VERSION)-$(shell go env GOOS)-$(shell go env GOARCH).tar.gz" -C dist/release "$(BIN)-$(VERSION)"
