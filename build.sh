#!/usr/bin/env bash
set -euo pipefail

go build -x -o build/mta-gtfs-probe ./cmd/mta-gtfs-probe.go
