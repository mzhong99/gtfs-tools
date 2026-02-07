#!/usr/bin/env bash
set -euo pipefail

go build -v -o build/mta-gtfs-probe ./cmd/mta-gtfs-probe.go
go build -v -o build/gtfs-ingest ./cmd/gtfs-ingest.go
