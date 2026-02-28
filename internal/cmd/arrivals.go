package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewArrivalsCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arrivals",
		Short: "Inspect recent trains arrivals",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Arrivals: NYI")
			return nil
		},
	}

	return cmd
}
