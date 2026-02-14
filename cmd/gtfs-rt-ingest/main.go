package main

import (
	"os"

	"tarediiran-industries.com/gtfs-services/internal/ingest/gtfs_rt"
)

func main() {
	os.Exit(gtfs_rt.Main(os.Args[0], os.Args[1:], os.Stdout, os.Stderr))
}
