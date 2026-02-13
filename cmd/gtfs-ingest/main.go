package main

import (
	"os"

	ingest "tarediiran-industries.com/gtfs-services/internal/ingest/gtfs_static"
)

func main() {
	os.Exit(ingest.Main(os.Args[0], os.Args[1:], os.Stdout, os.Stderr))
}
