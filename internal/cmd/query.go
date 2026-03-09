package cmd

import "github.com/spf13/cobra"

func NewQueryCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Subcommand for query commands",
	}

	cmd.AddCommand(NewArrivalsCmd(app))
	cmd.AddCommand(NewTripsCmd(app))
	cmd.AddCommand(NewStationsCmd(app))
	cmd.AddCommand(NewDebugQueryCmd(app))

	return cmd
}
