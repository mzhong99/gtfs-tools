package main

import (
	"os"
	"tarediiran-industries.com/gtfs-backend/internal/ingest"
)


func main() {
	os.Exit(ingest.Main(os.Args[0], os.Args[1:], os.Stdout, os.Stderr))
}
