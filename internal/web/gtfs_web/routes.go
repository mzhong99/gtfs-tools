package gtfs_web

import (
	"net/http"
	"time"
)

func (server *GtfsWebServer) handleTrainsPage(writer http.ResponseWriter, request *http.Request) {
	query := ParseTrainsQuery(request.URL.Query())

	routes := MockRoutes() // Replace with real routes getter later
	viewmodel := BuildTrainsPageVM(routes, query.Route, 2)

	writer.Header().Set("Content-Type", "text;html; charset=utf-8")
	if err := server.renderer.Render(writer, "layout.html", viewmodel); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (server *GtfsWebServer) handleTrainsPartial(writer http.ResponseWriter, request *http.Request) {
	query := ParseTrainsQuery(request.URL.Query())

	now := time.Now()
	trains := MockTrains(query.Route, now)
	viewmodel := BuildTrainsTableVM(query.Route, trains, now)

	writer.Header().Set("Content-Type", "text;html; charset=utf-8")
	if err := server.renderer.Render(writer, "trains_table.html", viewmodel); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}
