package gtfs_web

import (
	"fmt"
	"time"
)

type TrainDTO struct {
	Route     string
	Direction string
	TripID    string
	LastSeen  time.Time
	Status    string
}

func BuildTrainsPageVM(routes []string, selected string, pollSeconds int) TrainsPageVM {
	out := make([]string, 0, len(routes)+1)
	out = append(out, "ALL")
	out = append(out, routes...)

	if selected == "" {
		selected = "ALL"
	}
	if pollSeconds <= 0 {
		pollSeconds = 2
	}

	return TrainsPageVM{
		Routes:        out,
		SelectedRoute: selected,
		PollSeconds:   pollSeconds,
	}
}

func BuildTrainsTableVM(selected string, trains []TrainDTO, now time.Time) TrainsTableVM {
	rows := make([]TrainRowVM, 0, len(trains))
	for _, t := range trains {
		rows = append(rows, TrainRowVM{
			Route:     t.Route,
			Direction: t.Direction,
			TripID:    t.TripID,
			LastSeen:  formatAge(now, t.LastSeen),
			Status:    t.Status,
		})
	}

	return TrainsTableVM{
		SelectedRoute: selected,
		UpdatedAt:     now.Format("15:04:05"),
		Rows:          rows,
	}
}

func formatAge(now, then time.Time) string {
	d := now.Sub(then)
	if d < 0 {
		d = 0
	}
	// Keep it readable at a glance
	if d < 10*time.Second {
		return fmt.Sprintf("%.1fs ago", d.Seconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}
