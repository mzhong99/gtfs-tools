package gtfs_rt

import (
	"context"
	"fmt"

	"tarediiran-industries.com/gtfs-services/internal/platform"
)

type FileRecorder struct {
	pollerSet *PollerSet
	recording *platform.FeedRecordingWriter
	metrics   *platform.Metrics
	telemetry *platform.TelemetryServer
}

func WritePollResult(
	ctx context.Context, writer *platform.FeedRecordingWriter, result PollResult,
) error {
	frame := result.ToFeedFrame()
	fmt.Printf("Wrote: %s\n", frame)
	return writer.Append(ctx, frame)
}

func NewFileRecorder(
	ctx context.Context, config platform.SingleConfig, opts platform.RecordingHeaderOptions,
) (*FileRecorder, error) {
	feeds, err := config.Feed.ToFeedSpecs()
	if err != nil {
		return nil, err
	}

	recording, err := platform.CreateFeedRecording(feeds, opts)
	if err != nil {
		return nil, err
	}

	pollerSet, err := NewPollerSet(ctx, config)
	if err != nil {
		return nil, err
	}

	telemetry := config.NewTelemetryServer()
	metrics := platform.NewMetrics(telemetry.GetRegistry())

	recorder := &FileRecorder{
		pollerSet: pollerSet,
		recording: recording,
		telemetry: telemetry,
		metrics:   metrics,
	}

	return recorder, nil
}

func (recorder *FileRecorder) Record(ctx context.Context) error {
	recorder.pollerSet.SetHandler(func(ctx context.Context, result PollResult) error {
		return WritePollResult(ctx, recorder.recording, result)
	})

	recorder.telemetry.Start()
	return recorder.pollerSet.Poll(ctx)
}

func (recorder *FileRecorder) Stop() error {
	recorder.pollerSet.Stop()
	return recorder.telemetry.Stop()
}
