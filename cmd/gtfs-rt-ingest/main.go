package main

import (
	"os"

	"tarediiran-industries.com/gtfs-services/internal/ingest/gtfs_rt"
	"tarediiran-industries.com/gtfs-services/internal/platform"
)

func main() {
	os.Exit(platform.ParseArgsAndRun(os.Args, os.Stdout, os.Stderr, gtfs_rt.Run))
}
