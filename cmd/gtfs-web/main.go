package main

import (
	"os"

	"tarediiran-industries.com/gtfs-services/internal/web/gtfs_web"
)

func main() {
	os.Exit(gtfs_web.Main(os.Args[0], os.Args[1:], os.Stdout, os.Stderr))
}
