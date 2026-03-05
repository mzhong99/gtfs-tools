package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"tarediiran-industries.com/gtfs-services/internal/common"
	"tarediiran-industries.com/gtfs-services/internal/ingest/gtfs_rt"
)

func NewRecordCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record <recording_name>",
		Short: "Record live data from all supplied feeds into a location on disk.",
		RunE:  app.DoRecord,
		Args:  cobra.ExactArgs(1),
	}

	return cmd
}

func WritePollResult(
	ctx context.Context,
	writer *common.FeedRecordingWriter,
	result gtfs_rt.PollResult,
) error {
	frame := result.ToFeedFrame()
	fmt.Printf("Wrote: %s\n", frame)
	return writer.Append(ctx, frame)
}

func (app *GtfsCtlApp) DoRecord(cmd *cobra.Command, args []string) error {
	pollerSet, err := gtfs_rt.NewPollerSet(app.Context, app.Config)
	if err != nil {
		return err
	}

	now := time.Now()
	opts := common.RecordingHeaderOptions{
		RecordingName: args[0],
		CreatedAt:     now,
		TimeZone:      now.Location().String(),
		Tool:          common.NewToolInfo(cmd.Root().Name()),
	}

	feeds, err := app.Config.Feed.ToFeedSpecs()
	if err != nil {
		return err
	}

	recordingDir := filepath.Join(app.Layout.RecordingsDir, opts.RecordingName)
	recorder, err := common.CreateFeedRecording(recordingDir, feeds, opts)
	if err != nil {
		return err
	}

	pollerSet.SetHandler(func(ctx context.Context, result gtfs_rt.PollResult) error {
		return WritePollResult(ctx, recorder, result)
	})

	log.Println("================================================================================")
	log.Printf("Start file record to %s\n", opts.RecordingName)
	log.Println("================================================================================")
	log.Println(pollerSet)

	return pollerSet.Poll(app.Context)
}
