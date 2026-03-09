package cmd

import "github.com/spf13/cobra"

func NewCaptureCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "capture",
		Short:         "Commands to manage live captures of GTFS streams",
		SilenceErrors: false,
	}

	cmd.AddCommand(NewRecordCmd(app))
	cmd.AddCommand(NewPlaybackCmd(app))

	return cmd
}
