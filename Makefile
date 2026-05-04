.DEFAULT_GOAL := build

# Resolve the version string baked into the binary. Order:
#   1. $(VERSION) explicitly passed (`make build VERSION=v0.1.0`).
#   2. The current commit's exact tag, if HEAD is on a tag.
#   3. Empty — leave the runtime/debug fallback to do its thing.
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null)

PKG     := github.com/OysterD3/wtmux/internal/cli
LDFLAGS := $(if $(VERSION),-ldflags=-X $(PKG).version=$(VERSION))

BIN_DIR    := bin
BIN_NAME   := wtmux
BIN        := $(BIN_DIR)/$(BIN_NAME)
INSTALL_TO ?= $(HOME)/go/bin

.PHONY: build install test test-race lint vet clean release

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN) ./cmd/wtmux
	@echo "built $(BIN) ($$($(BIN) --version))"

# Install to a side-name (`wtmux-go`) by default so it doesn't clobber a
# pre-existing TS-built `wtmux` binary on PATH. Override INSTALL_NAME if
# you want a different name, or set it to `wtmux` once the TS version is
# fully retired.
INSTALL_NAME ?= wtmux-go
install: build
	@mkdir -p $(INSTALL_TO)
	cp $(BIN) $(INSTALL_TO)/$(INSTALL_NAME)
	@echo "installed $(INSTALL_TO)/$(INSTALL_NAME)"

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR) dist

# Cross-compile a tagged release into dist/. Run with VERSION set to the
# tag name (`make release VERSION=v0.1.0`); the GitHub Actions release
# workflow uses this target verbatim.
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64

release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release VERSION=v0.1.0)"; exit 1; fi
	@rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=dist/wtmux-$(VERSION)-$$os-$$arch; \
		echo "→ $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build -ldflags="-X $(PKG).version=$(VERSION) -s -w" -o $$out ./cmd/wtmux; \
		( cd dist && tar -czf $$(basename $$out).tar.gz $$(basename $$out) && rm $$(basename $$out) ); \
	done
	@( cd dist && shasum -a 256 *.tar.gz > checksums.txt )
	@echo "release artifacts in dist/"
