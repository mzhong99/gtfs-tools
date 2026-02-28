package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRoutesCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Inspect activity for transit routes",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Routes: NYI")
			return nil
		},
	}

	return cmd
}
