package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewStationsCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stations",
		Short: "Inspect state of stations",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Stations: NYI")
			return nil
		},
	}

	return cmd
}
