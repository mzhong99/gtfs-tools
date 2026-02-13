package ingest

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tarediiran-industries.com/gtfs-services/internal/db"
)

type FileTableEntry struct {
	FileName  string
	TableName string
	Required  bool
	Loader    func(context.Context, db.DBTX, string) error
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

func ReadCSVAsMapSlice(filePath string) ([]map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var records []map[string]string
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}

		record := make(map[string]string)
		if len(row) != len(headers) {
			return nil, fmt.Errorf("Row has different number of fields than headers")
		}

		for i, header := range headers {
			record[header] = row[i]
		}
		records = append(records, record)
	}
	return records, nil
}

func printSlices(slice []map[string]string) {
	js, err := json.MarshalIndent(slice, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(js))
}

func buildInsertQuery(tableName string, record map[string]string) (string, []any) {
	keys := make([]string, 0, len(record))
	values := make([]any, 0, len(record))
	placeholders := make([]string, 0, len(record))

	i := 1
	for col, val := range record {
		keys = append(keys, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	keysStr := strings.Join(keys, ", ")
	placeholdersStr := strings.Join(placeholders, ", ")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, keysStr, placeholdersStr)
	return query, values
}

func loadGeneric(ctx context.Context, db db.DBTX, filePath string, tableName string) error {
	fmt.Printf("Loading %s from %s\n", tableName, filePath)
	slices, err := ReadCSVAsMapSlice(filePath)
	if err != nil {
		return err
	}

	for _, slice := range slices {
		query, values := buildInsertQuery(tableName, slice)
		_, err := db.ExecContext(ctx, query, values...)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadCSVForColumnNames(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Gets just the first row, which contains the headers
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	return headers, nil
}

func loadGenericCopy(ctx context.Context, conn db.DBTX, filePath string, tableName string) error {
	columns, err := ReadCSVForColumnNames(filePath)
	if err != nil {
		return err
	}

	_, err = conn.(db.CopyCapable).CopyFrom(ctx, tableName, columns, filePath)
	return err
}

func loadAgency(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGeneric(ctx, db, filePath, "agency")
}

func loadRoutes(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGeneric(ctx, db, filePath, "routes")
}

func loadTrips(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGenericCopy(ctx, db, filePath, "trips")
}

func loadStops(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGenericCopy(ctx, db, filePath, "stops")
}

func loadStopTimes(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGenericCopy(ctx, db, filePath, "stop_times")
}

func loadCalendar(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGeneric(ctx, db, filePath, "calendar")
}

func loadCalendarDates(ctx context.Context, db db.DBTX, filePath string) error {
	return loadGeneric(ctx, db, filePath, "calendar_dates")
}

func loadShapes(ctx context.Context, db db.DBTX, filePath string) error {
	// return loadGeneric(ctx, db, filePath, "shapes")
	return nil
}

func loadTransfers(ctx context.Context, db db.DBTX, filePath string) error {
	// return loadGeneric(ctx, db, filePath, "transfers")
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

func LoadGtfsFromDirectory(dirPath string, domainStringName string) error {
	ctx := context.Background()
	db, err := db.NewDatabaseConnection(ctx, domainStringName)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, entry := range FileTableMapping {
		filePath := filepath.Join(dirPath, entry.FileName)
		ctx = context.Background()
		if entry.Loader == nil {
			continue
		}

		if err := entry.Loader(ctx, db, filePath); err != nil {
			return err
		}
	}

	return nil
}
