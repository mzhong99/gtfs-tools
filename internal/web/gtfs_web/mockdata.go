package gtfs_web

import (
	"slices"
	"time"
)

func MockRoutes() []string {
	routes := []string{"A", "C", "E", "1", "2", "3", "N", "Q", "R", "W", "L", "G"}
	slices.Sort(routes)
	return routes
}

func MockTrains(route string, now time.Time) []TrainDTO {
	all := []TrainDTO{
		{Route: "A", Direction: "N", TripID: "A..001", LastSeen: now.Add(-2 * time.Second), Status: "IN_TRANSIT"},
		{Route: "A", Direction: "S", TripID: "A..044", LastSeen: now.Add(-1 * time.Second), Status: "IN_TRANSIT"},
		{Route: "C", Direction: "N", TripID: "C..012", LastSeen: now.Add(-5 * time.Second), Status: "STOPPED"},
		{Route: "1", Direction: "S", TripID: "1..387", LastSeen: now.Add(-3 * time.Second), Status: "IN_TRANSIT"},
		{Route: "L", Direction: "E", TripID: "L..099", LastSeen: now.Add(-4 * time.Second), Status: "IN_TRANSIT"},
		{Route: "G", Direction: "S", TripID: "G..210", LastSeen: now.Add(-14 * time.Second), Status: "IN_TRANSIT"},
	}

	if route == "" || route == "ALL" {
		return all
	}

	out := make([]TrainDTO, 0, len(all))
	for _, t := range all {
		if t.Route == route {
			out = append(out, t)
		}
	}
	return out
}
