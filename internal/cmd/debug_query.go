package cmd

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func NewSqlReadQueryCmd(app *GtfsCtlApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sql <query>",
		Short: "Runs a manual PostgreSQL query. DANGEROUS - SQL Injections possible!!!",
		RunE:  app.DoSqlReadQuery,
		Args:  cobra.ExactArgs(1),
	}

	return cmd
}

func (app *GtfsCtlApp) DoSqlReadQuery(cmd *cobra.Command, args []string) error {
	return app.DoSqlReadQuerySafe(cmd, args[0], args[1:])
}

func (app *GtfsCtlApp) DoSqlReadQuerySafe(cmd *cobra.Command, query string, args []string) error {
	db, err := app.Config.NewDatabase(app.Context)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.QueryContext(app.Context, query, args)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	writer := table.NewWriter()
	header := make(table.Row, len(columns))
	for i, column := range columns {
		header[i] = column
	}
	writer.AppendHeader(header)

	dbRowValues := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	for i := range dbRowValues {
		scanTargets[i] = &dbRowValues[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanTargets...); err != nil {
			return err
		}

		row := make(table.Row, len(columns))
		for i, value := range dbRowValues {
			row[i] = formatDbValue(value)
		}
		writer.AppendRow(row)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	fmt.Println(writer.Render())
	return nil
}

func formatDbValue(v any) any {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	default:
		return x
	}
}
