package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"tarediiran-industries.com/gtfs-services/internal/ingest/gtfs_rt"
	"tarediiran-industries.com/gtfs-services/internal/platform"
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
	writer *platform.FeedRecordingWriter,
	result gtfs_rt.PollResult,
) error {
	frame := result.ToFeedFrame()
	fmt.Printf("Wrote: %s\n", frame)
	return writer.Append(ctx, frame)
}

func (app *GtfsCtlApp) DoRecord(cmd *cobra.Command, args []string) error {
	now := time.Now()
	opts := platform.RecordingHeaderOptions{
		RecordingName: args[0],
		RecordingPath: app.Layout.RecordingsDir,
		CreatedAt:     now,
		TimeZone:      now.Location().String(),
		Tool:          platform.NewToolInfo(cmd.Root().Name()),
	}

	recorder, err := gtfs_rt.NewFileRecorder(app.Context, app.Config, opts)
	if err != nil {
		return err
	}

	log.Printf("================================================================================\n")
	log.Printf("Start file record: %s\n", opts.GetRecordingPath())
	log.Printf("================================================================================\n")

	return recorder.Record(app.Context)
}
