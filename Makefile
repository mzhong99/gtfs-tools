MODULE := $(shell awk '$$1=="module"{print $$2}' go.mod)

VERSION := $(shell git describe --tags --dirty --always)
COMMIT := $(shell git rev-parse --short HEAD)

PKG_VERSION := $(MODULE)/internal/common

LDFLAGS := \
  -X $(PKG_VERSION).Version=$(VERSION) \
  -X $(PKG_VERSION).GitCommit=$(COMMIT)

BINS := \
    gtfs-ingest \
    gtfs-rt-ingest \
    gtfs-web

all: build

build:
	@mkdir -p bin
	@for b in $(BINS); do \
		echo "[BUILD] $$b"; \
		go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$$b ./cmd/$$b ; \
	done

format:
	go fmt ./...

clean:
	@echo [CLEAN] bin
	@rm -rf bin

.PHONY: infra-up infra-down db-reset db-migrate-up db-migrate-down db-migrate-status

infra-up:
	docker-compose up -d postgres prometheus
	@echo "Waiting for postgres to be ready..."
	@until [ "$$(docker inspect --format='{{.State.Health.Status}}' $$(docker-compose ps -q postgres))" = "healthy" ]; do \
		sleep 0.2; \
	done
	@echo "Waiting for prometheus to be ready..."
	@until curl -fsS http://localhost:9090/-/ready >/dev/null; do \
		sleep 0.2; \
	done

infra-down:
	docker-compose down -v

db-reset: infra-down infra-up db-migrate-up
	@echo "Database reset complete"

db-migrate-up: infra-up
	migrate -verbose -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" up

db-migrate-down:
	migrate -verbose -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" down

db-migrate-status:
	migrate -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" version
