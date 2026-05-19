VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.Date=$(DATE)

.PHONY: build test vet install install-local snapshot clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/ferry ./cmd/ferry

test:
	go test ./...

vet:
	go vet ./...

install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/ferry

# install-local builds a production-style binary and places it on PATH via
# scripts/install.sh (defaults to ~/.local/bin; override with INSTALL_DIR=...).
install-local:
	./scripts/install.sh

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/
