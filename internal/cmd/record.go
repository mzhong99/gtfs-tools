package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRecordCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "Record live data from all supplied feeds into a location on disk.",
		RunE:  app.DoRecord,
	}

	return cmd
}

func (app *GtfsCtlApp) DoRecord(cmd *cobra.Command, args []string) error {
	err := app.ParseConfig()
	fmt.Printf("Config: %v\n", app.Config)
	return err
}
