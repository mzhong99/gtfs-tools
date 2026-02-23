package gtfs_web

import (
	"net/url"
	"strings"
)

type TrainsQuery struct {
	Route string // "ALL" or specific route
}

func ParseTrainsQuery(values url.Values) TrainsQuery {
	route := strings.TrimSpace(values.Get("route"))
	route = strings.ToUpper(route)
	if route == "" {
		route = "ALL"
	}
	return TrainsQuery{Route: route}
}
