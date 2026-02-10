package ingest

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileTableEntry struct {
	FileName  string
	TableName string
	Required  bool
	Loader    func(string) error
}

var FileTableMapping = []FileTableEntry{
	{FileName: "agency.txt", TableName: "agency", Required: true, Loader: loadAgency},
	{FileName: "routes.txt", TableName: "routes", Required: true, Loader: loadRoutes},
	{FileName: "trips.txt", TableName: "trips", Required: true, Loader: loadTrips},
	{FileName: "stops.txt", TableName: "stops", Required: true, Loader: loadStops},
	{FileName: "stop_times.txt", TableName: "stop_times", Required: true, Loader: loadStopTimes},
	{FileName: "calendar.txt", TableName: "calendar", Required: true, Loader: loadCalendar},
	{FileName: "calendar_dates.txt", TableName: "calendar_dates", Required: true, Loader: loadCalendarDates},
	{FileName: "shapes.txt", TableName: "shapes", Required: false, Loader: loadShapes},
	{FileName: "transfers.txt", TableName: "transfers", Required: false, Loader: loadTransfers},
}

func loadAgency(filePath string) error {
	return nil
}

func loadRoutes(filePath string) error {
	return nil
}

func loadTrips(filePath string) error {
	return nil
}

func loadStops(filePath string) error {
	return nil
}

func loadStopTimes(filePath string) error {
	return nil
}

func loadCalendar(filePath string) error {
	return nil
}

func loadCalendarDates(filePath string) error {
	return nil
}

func loadShapes(filePath string) error {
	return nil
}

func loadTransfers(filePath string) error {
	return nil
}

func ValidateGtfsDirectory(dirPath string) error {
	for _, entry := range FileTableMapping {
		if entry.Required {
			filePath := filepath.Join(dirPath, entry.FileName)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("Required file %s is missing", entry.FileName)
			}
		}
	}

	return nil
}
