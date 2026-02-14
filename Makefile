MODULE := $(shell awk '$$1=="module"{print $$2}' go.mod)

VERSION := $(shell git describe --tags --dirty --always)
COMMIT := $(shell git rev-parse --short HEAD)

PKG_VERSION := $(MODULE)/internal/common

LDFLAGS := \
  -X $(PKG_VERSION).Version=$(VERSION) \
  -X $(PKG_VERSION).GitCommit=$(COMMIT)

BINS := \
    gtfs-ingest \
    gtfs-rt-ingest

all: build

build:
	@mkdir -p bin
	@for b in $(BINS); do \
		echo "[BUILD] $$b"; \
		go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$$b ./cmd/$$b ; \
	done

clean:
	@echo [CLEAN] bin
	@rm -rf bin

.PHONY: db-up db-down db-reset db-migrate-up db-migrate-down db-migrate-status

db-up:
	docker-compose up -d postgres
	@echo "Waiting for postgres to be ready..."
	@until [ "$$(docker inspect --format='{{.State.Health.Status}}' $$(docker-compose ps -q postgres))" = "healthy" ]; do \
		sleep 0.2; \
	done

db-down:
	docker-compose down -v

db-reset: db-down db-up db-migrate-up
	@echo "Database reset complete"

db-migrate-up: db-up
	migrate -verbose -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" up

db-migrate-down:
	migrate -verbose -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" down

db-migrate-status:
	migrate -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" version
