package main

import (
	"tarediiran-industries.com/gtfs-services/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
