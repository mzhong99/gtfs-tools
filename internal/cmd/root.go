package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"tarediiran-industries.com/gtfs-services/internal/platform"
)

type GtfsCtlApp struct {
	ConfigPath string
	Config     platform.SingleConfig
	Context    context.Context
	Layout     platform.PlatformLayout
}

func (app *GtfsCtlApp) ParseConfig() error {
	var err error
	app.Config, err = platform.LoadConfigFromToml(app.ConfigPath)
	if err != nil {
		return err
	}
	return nil
}

func (app *GtfsCtlApp) Init() error {
	root, err := platform.DefaultPlatformRoot()
	if err != nil {
		return err
	}

	app.Layout = platform.NewPlatformLayout(root)
	if err := platform.InitPlatform(app.Layout); err != nil {
		return err
	}

	return app.ParseConfig()
}

func Execute() error {
	app := &GtfsCtlApp{}
	var stop context.CancelFunc
	app.Context, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := NewRootCmd(app)
	return rootCmd.Execute()
}

func NewRootCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "gtfs-ctl",
		Short:             "CLI tool used to inspect GTFS static and real-time state",
		SilenceUsage:      true,
		SilenceErrors:     false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return app.Init() },
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
	cmd.AddCommand(NewRecordCmd(app))
	cmd.AddCommand(NewPlaybackCmd(app))

	return cmd
}
