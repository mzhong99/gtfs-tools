package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewTripsCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trips",
		Short: "Inspect scheduled and upcoming train trips",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Trips: NYI")
			return nil
		},
	}

	return cmd
}
