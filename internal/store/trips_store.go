package store

import (
	"fmt"
	"strings"

	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type TripsQuery struct {
	Routes          []string
	Direction       int
	Service         string
	Pattern         string
	IncludeStopTime bool
	HeadSign        string
	StopIDs         []string
}

func (query TripsQuery) IsDefault() bool {
	return len(query.Routes) == 0 &&
		query.Direction == -1 &&
		query.Service == "" &&
		query.Pattern == "" &&
		query.HeadSign == "" &&
		len(query.StopIDs) == 0
}

type TripsRow struct {
	TripID       string
	RouteID      string
	ServiceID    string
	TripHeadsign string
	DirectionID  string
}

func ListTrips(db *database.Database, query TripsQuery) ([]TripsRow, error) {
	queryString, queryArgs := buildSqlString(query)
	fmt.Println(queryString)
	fmt.Println(queryArgs)
	return nil, nil
}

func buildSqlString(query TripsQuery) (string, []any) {
	var b strings.Builder

	b.WriteString(`
		select
			trip_id, route_id, service_id, trip_headsign, direction_id
		from
			trips`,
	)

	if query.IncludeStopTime {
		b.WriteString("join stop_times on trips.trip_id = stop_times.trip_id")
	}
	b.WriteString("\n")

	if query.IsDefault() {
		return b.String(), nil
	}

	args := make([]any, 0)
	b.WriteString("where")

	sep := "\n"
	if len(query.Routes) > 0 {
		b.WriteString(fmt.Sprintf("%s (", sep))
		innerSep := ""
		for _, route := range query.Routes {
			args = append(args, route)
			b.WriteString(fmt.Sprintf("%s route_id = $%d", innerSep, len(args)))
			innerSep = " or"
		}
		b.WriteString(")")
		sep = "\nand "
	}
	if query.Direction != -1 {
		args = append(args, query.Direction)
		b.WriteString(fmt.Sprintf("%s direction_id = $%d", sep, len(args)))
		sep = "\nand "
	}
	if query.Service != "" {
		args = append(args, query.Service)
		b.WriteString(fmt.Sprintf("%s service_id = $%d", sep, len(args)))
		sep = "\nand "
	}
	if query.Pattern != "" {
		args = append(args, query.Pattern)
		b.WriteString(fmt.Sprintf("%s pattern_id = $%d", sep, len(args)))
		sep = "\nand "
	}
	if query.HeadSign != "" {
		b.WriteString(fmt.Sprintf("%s trip_headsign = $%d", sep, len(args)))
		b.WriteString(sep)
		sep = "\nand "
	}
	if len(query.StopIDs) > 0 {
		b.WriteString(fmt.Sprintf("%s (", sep))
		innerSep := ""
		for _, stopID := range query.StopIDs {
			args = append(args, stopID)
			b.WriteString(fmt.Sprintf("%s stop_id = $%d", innerSep, len(args)))
			innerSep = " or"
		}
		b.WriteString(")")
		sep = "\nand "
	}

	return b.String(), args
}
