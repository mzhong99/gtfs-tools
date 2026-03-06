package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"tarediiran-industries.com/gtfs-services/internal/ingest/gtfs_rt"
)

func NewPlaybackCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playback <recording_name>",
		Short: "Playback a selected recording, providing feed recordings for data ingest",
		RunE:  app.DoPlayback,
		Args:  cobra.ExactArgs(1),
	}

	cmd.Flags().Bool("delays", false, "Respect recorded real-time delays")
	cmd.Flags().Bool("list", false, "Respect recorded real-time delays")

	return cmd
}

func (app *GtfsCtlApp) listRecordings() error {
	fmt.Println("List NYI")
	return nil
}

func (app *GtfsCtlApp) DoPlayback(cmd *cobra.Command, args []string) error {
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return err
	}
	if list {
		return app.listRecordings()
	}

	delays, err := cmd.Flags().GetBool("delays")
	if err != nil {
		return err
	}

	recordingName := args[0]
	recordingPath := filepath.Join(app.Layout.RecordingsDir, recordingName)
	playback, err := gtfs_rt.NewFilePlayback(app.Context, app.Config, recordingPath)
	if err != nil {
		return err
	}
	defer playback.Close()

	if delays {
		return playback.PlaybackRealtime(app.Context)
	}
	return playback.PlaybackFast(app.Context)
}
