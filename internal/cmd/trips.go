package cmd

import (
	"github.com/spf13/cobra"
	"tarediiran-industries.com/gtfs-services/internal/store"
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

	cmd.Flags().StringSlice("route", nil, "Filter by route ID (1, 2, 3, A, C, E, B, D, F...)")
	cmd.Flags().Int("direction", -1, "Filter by direction ID (0, 1)")
	cmd.Flags().String("service", "", "Filter by service ID (Saturday, Sunday, Weekday...)")
	cmd.Flags().String("pattern", "", "Filter by pattern ID")
	cmd.Flags().Bool("with-stop-time", false, "List anticipated stop times for a trip")
	cmd.Flags().String("headsign", "", "Filter by trip headsign")
	cmd.Flags().StringSlice("contains-stop", nil, "Filter by stop ID")

	return cmd
}

func (app *GtfsCtlApp) DoTripsList(cmd *cobra.Command, args []string) error {
	var err error

	db, err := app.Config.NewDatabase(app.Context)
	if err != nil {
		return err
	}
	defer db.Close()

	routes, err := cmd.Flags().GetStringSlice("route")
	if err != nil {
		return err
	}
	direction, err := cmd.Flags().GetInt("direction")
	if err != nil {
		return err
	}
	service, err := cmd.Flags().GetString("service")
	if err != nil {
		return err
	}
	pattern, err := cmd.Flags().GetString("pattern")
	if err != nil {
		return err
	}
	includeStopTime, err := cmd.Flags().GetBool("with-stop-time")
	if err != nil {
		return err
	}
	headsign, err := cmd.Flags().GetString("headsign")
	if err != nil {
		return err
	}
	stopIDs, err := cmd.Flags().GetStringSlice("contains-stop")
	if err != nil {
		return err
	}

	query := store.TripsQuery{
		Routes:          routes,
		Direction:       direction,
		Service:         service,
		Pattern:         pattern,
		IncludeStopTime: includeStopTime,
		HeadSign:        headsign,
		StopIDs:         stopIDs,
	}

	store.ListTrips(db, query)

	return nil
}
