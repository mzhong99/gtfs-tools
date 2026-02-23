package gtfs_web

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func Run(cfg Config) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := NewGtfsWebServer(ctx, cfg.ListenAddress, cfg.DatabaseConnection)
	if err != nil {
		panic(err)
	}

	server.Serve(ctx)
	return 0
}
