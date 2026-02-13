package ingest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tarediiran-industries.com/gtfs-services/internal/db"
	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type Ingestor struct {
	feedId int
	db     database.DBTX
	ctx    context.Context
}

type FileTableEntry struct {
	FileName  string
	TableName string
	Required  bool
	Loader    func(Ingestor, string) error
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

func (ingestor *Ingestor) loadGeneric(filePath string, tableName string) error {
	fmt.Printf("Loading %s from %s\n", tableName, filePath)
	slices, err := ReadCSVAsMapSlice(filePath)
	if err != nil {
		return err
	}

	for _, slice := range slices {
		query, values := buildInsertQuery(tableName, slice)
		_, err := ingestor.db.ExecContext(ingestor.ctx, query, values...)
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

func (ingestor *Ingestor) loadGenericCopy(filePath string, tableName string) error {
	columns, err := ReadCSVForColumnNames(filePath)
	if err != nil {
		return err
	}

	_, err = ingestor.db.(db.CopyCapable).CopyFrom(ingestor.ctx, tableName, columns, filePath)
	return err
}

func loadAgency(ingestor Ingestor, filePath string) error {
	return ingestor.loadGeneric(filePath, "agency")
}

func loadRoutes(ingestor Ingestor, filePath string) error {
	return ingestor.loadGeneric(filePath, "routes")
}

func loadTrips(ingestor Ingestor, filePath string) error {
	return ingestor.loadGenericCopy(filePath, "trips")
}

func loadStops(ingestor Ingestor, filePath string) error {
	return ingestor.loadGenericCopy(filePath, "stops")
}

func loadStopTimes(ingestor Ingestor, filePath string) error {
	return ingestor.loadGenericCopy(filePath, "stop_times")
}

func loadCalendar(ingestor Ingestor, filePath string) error {
	return ingestor.loadGeneric(filePath, "calendar")
}

func loadCalendarDates(ingestor Ingestor, filePath string) error {
	return ingestor.loadGeneric(filePath, "calendar_dates")
}

func loadShapes(ingestor Ingestor, filePath string) error {
	// return loadGeneric(ingestor.ctx, ingestor.db, filePath, "shapes")
	return nil
}

func loadTransfers(ingestor Ingestor, filePath string) error {
	// return loadGeneric(ingestor.ctx, ingestor.db, filePath, "transfers")
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

func StartFeedIngest(ctx context.Context, db db.DBTX, urlPath string, zipPath string) (int, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return 0, fmt.Errorf("failed to hash file: %w", err)
	}
	hashSum := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashSum)

	row := db.QueryRowContext(ctx, `
		INSERT INTO feed_version (imported_at, source_url, source_sha256)
		VALUES (NOW(), $1, $2)
		RETURNING feed_id
	`, urlPath, hashHex)

	var feedId int
	if err := row.Scan(&feedId); err != nil {
		return 0, fmt.Errorf("failed to scan feed_id: %w", err)
	}

	return feedId, nil
}

func FeedExistsForHash(ctx context.Context, db db.DBTX, zipPath string) (bool, error) {
	file, err := os.Open(zipPath)
	if err != nil {
		fmt.Printf("Failed to open file for hashing: %v\n", err)
		return false, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		fmt.Printf("Failed to hash file: %v\n", err)
		return false, err
	}
	hashSum := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashSum)
	fmt.Printf("hashHex: %v\n", hashHex)

	existingFeedId := 0
	err = db.QueryRowContext(ctx, `
		SELECT feed_id FROM feed_version
		WHERE source_sha256 = $1
	`, hashHex).Scan(&existingFeedId)

	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error checking for existing feed: %v\n", err)
		return false, err
	}

	return existingFeedId != 0, nil
}

func LoadGtfsFromDirectory(urlPath string, zipPath string, domainStringName string) error {
	extractDir, err := UnzipToTempDir(zipPath)
	if err != nil {
		return err
	}

	if err := ValidateGtfsDirectory(extractDir); err != nil {
		return err
	}

	ctx := context.Background()
	db, err := db.NewDatabaseConnection(ctx, domainStringName)
	if err != nil {
		return err
	}
	defer db.Close()

	if exists, err := FeedExistsForHash(ctx, db, zipPath); err != nil {
		return err
	} else if exists {
		fmt.Printf("GTFS data for %s has already been ingested.\n", zipPath)
		return nil
	}

	feedId, err := StartFeedIngest(ctx, db, urlPath, zipPath)
	if err != nil {
		return err
	}

	ingestor := Ingestor{feedId: feedId, db: db, ctx: ctx}
	for _, entry := range FileTableMapping {
		filePath := filepath.Join(extractDir, entry.FileName)
		ctx = context.Background()
		if entry.Loader == nil {
			continue
		}

		if err := entry.Loader(ingestor, filePath); err != nil {
			return err
		}
	}

	return nil
}
