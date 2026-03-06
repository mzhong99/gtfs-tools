package gtfs_rt

import (
	"context"
	"fmt"
	"io"
	"time"

	"tarediiran-industries.com/gtfs-services/internal/platform"
)

type PlaybackCallback func(ctx context.Context, frame platform.FeedFrame) error

type FilePlayback struct {
	recording   *platform.FeedRecordingReader
	ingesterSet *FeedIngesterSet
	handlers    map[string]PlaybackCallback

	metrics   *platform.Metrics
	telemetry *platform.TelemetryServer
}

func NewFilePlayback(
	ctx context.Context, config platform.SingleConfig, recordingDir string,
) (*FilePlayback, error) {
	var err error
	recording, err := platform.OpenFeedRecording(recordingDir)
	if err != nil {
		return nil, err
	}

	telemetry := config.NewTelemetryServer()
	metrics := platform.NewMetrics(telemetry.GetRegistry())

	ingesterSet, err := NewFeedIngesterSet(ctx, config)
	if err != nil {
		return nil, err
	}

	playback := &FilePlayback{
		recording:   recording,
		ingesterSet: ingesterSet,
		handlers:    make(map[string]PlaybackCallback),

		metrics:   metrics,
		telemetry: telemetry,
	}

	for i, _ := range ingesterSet.ingesters {
		ingester := &ingesterSet.ingesters[i]
		playback.SetHandler(
			ingester.cfg.ID,
			func(ctx context.Context, frame platform.FeedFrame) error {
				ingester := ingester
				return ingester.Ingest(ctx, frame)
			},
		)
	}

	playback.telemetry.Start()
	return playback, nil
}

func (playback *FilePlayback) SetHandler(feedId string, callback PlaybackCallback) {
	playback.handlers[feedId] = callback
}

func (playback *FilePlayback) PlaybackRealtime(ctx context.Context) error {
	return playback.doPlayback(ctx, true)
}

func (playback *FilePlayback) PlaybackFast(ctx context.Context) error {
	return playback.doPlayback(ctx, false)
}

func (playback *FilePlayback) doPlayback(ctx context.Context, doRealTimeDelays bool) error {
	lastCapturedAt := time.Time{}

	for {
		frame, err := playback.recording.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		callback, ok := playback.handlers[frame.FeedID]
		if !ok {
			fmt.Printf("Unable to service feed %s (unsupported in config)\n", frame.FeedID)
			continue
		}

		if err := callback(ctx, frame); err != nil {
			return err
		}

		if !lastCapturedAt.IsZero() && doRealTimeDelays {
			delay := frame.CapturedAt.Sub(lastCapturedAt)
			if delay > 0 {
				time.Sleep(delay)
			}
		}

		lastCapturedAt = frame.CapturedAt
	}

	return nil
}

func (playback *FilePlayback) Close() error {
	playback.ingesterSet.Stop()
	if err := playback.telemetry.Stop(); err != nil {
		return err
	}
	if err := playback.recording.Close(); err != nil {
		return err
	}
	return nil
}
