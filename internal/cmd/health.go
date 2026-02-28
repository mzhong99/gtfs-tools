package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewHealthCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Inspect health of GTFS backend services",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Health: NYI")
			return nil
		},
	}

	return cmd
}
