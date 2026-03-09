package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewTripsCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trips",
		Short: "Subcommand to inspect scheduled and upcoming train trips",
	}

	cmd.AddCommand(NewTripsListCmd(app))

	return cmd
}

func NewTripsListCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List all static trips (i.e. non-realtime) specified in this current GTFS instance",
		SilenceErrors: false,
		RunE:          app.DoTripsList,
	}

	cmd.Flags().String("route", "", "Filter by route ID (1, 2, 3, A, C, E, B, D, F...)")
	cmd.Flags().Int("direction", -1, "Filter by direction ID (0, 1)")
	cmd.Flags().String("service", "", "Filter by service ID (Saturday, Sunday, Weekday...)")
	cmd.Flags().String("pattern", "", "Filter by pattern ID")
	cmd.Flags().Bool("with-stop-time", false, "List anticipated stop times for a trip")
	cmd.Flags().String("headsign", "", "Filter by trip headsign")
	cmd.Flags().String("contains-stop", "", "Filter by stop ID")

	return cmd
}

func (app *GtfsCtlApp) DoTripsList(cmd *cobra.Command, args []string) error {
	fmt.Println("ayy lmao")
	return nil
}
