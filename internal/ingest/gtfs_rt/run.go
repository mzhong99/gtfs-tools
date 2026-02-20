package gtfs_rt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func Run(cfg Config) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	watcher, err := NewGtfsRtWatcher(ctx, cfg.Urls, cfg.DatabaseConnection, 1.0)
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	watcher.Watch(ctx)
	fmt.Println("Finished.")

	return 0
}
