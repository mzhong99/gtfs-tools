package cmd

import "github.com/spf13/cobra"

type GtfsCtlApp struct {
	ConfigPath string
}

func Execute() error {
	app := &GtfsCtlApp{}
	rootCmd := NewRootCmd(app)
	return rootCmd.Execute()
}

func NewRootCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gtfs-ctl",
		Short:         "CLI tool used to inspect GTFS static and real-time state",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	cmd.PersistentFlags().StringVar(
		&app.ConfigPath,
		"toml",
		"config/gtfs-mta.dev.toml",
		"Path to configuration file",
	)

	cmd.AddCommand(NewArrivalsCmd(app))
	cmd.AddCommand(NewTripsCmd(app))
	cmd.AddCommand(NewStationsCmd(app))

	return cmd
}
