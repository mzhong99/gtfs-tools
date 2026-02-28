MODULE := $(shell awk '$$1=="module"{print $$2}' go.mod)

VERSION := $(shell git describe --tags --dirty --always)
COMMIT := $(shell git rev-parse --short HEAD)
COMPOSE = DOCKER_BUILDKIT=0 docker compose


PKG_VERSION := $(MODULE)/internal/common

LDFLAGS := \
  -X $(PKG_VERSION).Version=$(VERSION) \
  -X $(PKG_VERSION).GitCommit=$(COMMIT)

BINS := \
    gtfs-ingest \
    gtfs-rt-ingest \
	gtfs-ctl \
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

.PHONY: container-build infra-up infra-down db-reset db-migrate-up db-migrate-down db-migrate-status

container-build:
	$(COMPOSE) --profile app build

infra-up: container-build
	$(COMPOSE) --profile infra up -d
	@echo "Waiting for postgres to be ready..."
	@until [ "$$(docker inspect --format='{{.State.Health.Status}}' $$($(COMPOSE) ps -q postgres))" = "healthy" ]; do \
		sleep 0.2; \
	done
	$(MAKE) db-migrate-up

app-up: container-build
	$(COMPOSE) --profile infra --profile app up -d

up: container-build
	$(MAKE) infra-up
	$(MAKE) app-up

down:
	$(MAKE) app-down
	$(MAKE) infra-down

infra-down:
	$(COMPOSE) --profile infra down

app-down:
	$(COMPOSE) --profile app down

db-reset:
	$(COMPOSE) --profile infra --profile app down -v
	@echo "Database reset complete"

db-migrate-up:
	migrate -verbose -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" up

db-migrate-status:
	migrate -path migrations -database "postgres://gtfs:gtfs_dev_password@localhost:5432/gtfs_db?sslmode=disable" version
