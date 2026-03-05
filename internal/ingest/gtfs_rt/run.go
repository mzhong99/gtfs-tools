package gtfs_rt

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"tarediiran-industries.com/gtfs-services/internal/platform"
)

func Run(cfg platform.SingleConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	watcher, err := NewWatcher(ctx, cfg)
	if err != nil {
		return err
	}
	defer watcher.Close()
	watcher.Watch(ctx)

	return nil
}
