package gtfs_rt

import (
	"context"

	"tarediiran-industries.com/gtfs-services/internal/platform"
)

type Watcher struct {
	cfg platform.SingleConfig

	pollerSet   *PollerSet
	ingesterSet *FeedIngesterSet

	metrics   *platform.Metrics
	telemetry *platform.TelemetryServer
}

func NewWatcher(ctx context.Context, cfg platform.SingleConfig) (*Watcher, error) {
	telemetry := cfg.NewTelemetryServer()
	telemetry.Start()
	metrics := platform.NewMetrics(telemetry.GetRegistry())

	pollerSet, err := NewPollerSet(ctx, cfg)
	if err != nil {
		return nil, err
	}

	ingesterSet, err := NewFeedIngesterSet(ctx, cfg)
	if err != nil {
		return nil, err
	}

	for _, ingester := range ingesterSet.ingesters {
		pollerSet.SetHandlerByID(
			ingester.cfg.ID,
			func(ctx context.Context, result PollResult) error {
				ingester := ingester
				return ingester.Ingest(ctx, result.ToFeedFrame())
			},
		)
	}

	return &Watcher{
		cfg:         cfg,
		pollerSet:   pollerSet,
		ingesterSet: ingesterSet,
		metrics:     metrics,
		telemetry:   telemetry,
	}, nil
}

func (watcher *Watcher) Watch(ctx context.Context) error {
	return watcher.pollerSet.Poll(ctx)
}

func (watcher *Watcher) Close() {
	watcher.telemetry.Stop()
	watcher.pollerSet.Stop()
	watcher.ingesterSet.Stop()
}
