package gtfs_rt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tarediiran-industries.com/gtfs-services/internal/common"
)

func Run(cfg common.SingleConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	urls := make([]string, 0)
	for _, feed := range cfg.Feed.RealTime {
		urls = append(urls, feed.URL)
	}
	watcher, err := NewGtfsRtWatcher(
		ctx, cfg.Observability.TelemetryUrl, urls, cfg.Database.URL, 1.0,
	)
	if err != nil {
		return err
	}
	defer watcher.Close()

	watcher.Watch(ctx)
	fmt.Println("Finished.")

	return nil
}
