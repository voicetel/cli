# voicetel-cli — cross-platform build matrix.
#
# Pure-Go (CGO_ENABLED=0) so cross-compilation works without per-target
# C toolchains. None of the SDK / readline / TOML deps need cgo.
#
# Targets:
#   make build       Build for the local platform → ./bin/voicetel-cli
#   make build-all   Cross-compile for every supported (OS, arch) → ./dist/
#   make release     build-all + per-platform archives (.tar.gz / .zip)
#   make test        go test ./...
#   make vet         go vet ./...
#   make install     CGO_ENABLED=0 go install . (lands in $$GOPATH/bin)
#   make clean       Remove ./bin and ./dist
#   make version     Print the version Makefile sees
#
# The binary name is voicetel-cli on every platform.

BINARY  := voicetel-cli
VERSION := $(shell awk -F'"' '/const Version/ {print $$2}' version.go)
LDFLAGS := -s -w

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	linux/386 \
	linux/arm \
	windows/amd64 \
	windows/arm64 \
	freebsd/amd64

.PHONY: help build build-all release test vet install clean version

help:
	@echo "voicetel-cli $(VERSION) — make targets:"
	@echo "  build       Build the binary for the local platform (./bin/$(BINARY))"
	@echo "  build-all   Cross-compile for $(words $(PLATFORMS)) platforms into ./dist/"
	@echo "  release     build-all + per-platform archive (.tar.gz / .zip)"
	@echo "  test        Run go test ./..."
	@echo "  vet         Run go vet ./..."
	@echo "  install     CGO_ENABLED=0 go install . (lands in \$$GOPATH/bin)"
	@echo "  clean       Remove ./bin and ./dist"
	@echo "  version     Print $(VERSION)"

version:
	@echo $(VERSION)

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) .
	@echo "→ bin/$(BINARY) ($(VERSION))"

test:
	go test ./...

vet:
	go vet ./...

install:
	CGO_ENABLED=0 go install -ldflags="$(LDFLAGS)" .

clean:
	rm -rf bin dist

# Build one binary per (OS, arch) under ./dist/voicetel-cli_<version>_<os>-<arch>/
build-all: clean
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		dir="dist/$(BINARY)_$(VERSION)_$$os-$$arch"; \
		echo "→ $$dir/$(BINARY)$$ext"; \
		mkdir -p "$$dir"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags="$(LDFLAGS)" \
			-o "$$dir/$(BINARY)$$ext" . || exit 1; \
		cp LICENSE README.md "$$dir/" 2>/dev/null || true; \
	done
	@echo "Built $(words $(PLATFORMS)) binaries in ./dist/"

# Package each platform build as tar.gz (or .zip for Windows) for GH release upload.
release: build-all
	@cd dist && for d in $(BINARY)_$(VERSION)_*/; do \
		base=$$(basename $$d); \
		case "$$base" in \
			*windows*) zip -qr "$$base.zip" "$$base" && echo "📦 $$base.zip" ;; \
			*)         tar czf "$$base.tar.gz" "$$base" && echo "📦 $$base.tar.gz" ;; \
		esac; \
	done
	@echo "Release archives ready in ./dist/"
