MODULE := $(shell awk '$$1=="module"{print $$2}' go.mod)

VERSION := $(shell git describe --tags --dirty --always)
COMMIT := $(shell git rev-parse --short HEAD)

PKG_VERSION := $(MODULE)/internal/common

LDFLAGS := \
  -X $(PKG_VERSION).Version=$(VERSION) \
  -X $(PKG_VERSION).GitCommit=$(COMMIT)

BINS := \
    gtfs-ingest

all: build

build:
	@mkdir -p bin
	@for b in $(BINS); do \
		echo "→ building $$b"; \
		go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$$b ./cmd/$$b ; \
	done

clean:
	rm -rf bin

