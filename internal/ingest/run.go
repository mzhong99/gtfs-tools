package ingest

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var GtfsRequiredFiles = []string{
	"agency.txt",
	"routes.txt",
	"trips.txt",
	"stops.txt",
	"stop_times.txt",
	"calendar.txt",
	"calendar_dates.txt",
}

func unzipToTempDir(zipPath string) (string, error) {
	dir, err := os.MkdirTemp("", "gtfs-ingest-*")
	if err != nil {
		return "", err
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	searchFiles := map[string]bool{}
	for _, requiredFile := range GtfsRequiredFiles {
		searchFiles[requiredFile] = false
	}

	// Walk zip files - extract contents to temp dir
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			return "", fmt.Errorf("Bro why is this a dir: %s", file.Name)
		}

		if _, ok := searchFiles[file.Name]; ok {
			searchFiles[file.Name] = true
		} else {
			// return "", fmt.Errorf("Unrecognized file %s", file.Name)
			fmt.Printf("Unrecognized file %s - ignore for now\n", file.Name)
			continue
		}

		dstPath := filepath.Join(dir, filepath.Base(file.Name))
		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := file.Open()
		if err != nil {
			panic(err)
		}

		bytesWritten, err := io.Copy(dstFile, fileInArchive)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Extracted %s -> %s (%d bytes)\n", file.Name, dstPath, bytesWritten)
	}

	return dir, nil
}

func Run(cfg Config) int {
	if cfg.Url != "" {
		fmt.Println("NYI: Web download")
	}

	var zipPath string = cfg.ZipPath
	tempDir, err := unzipToTempDir(zipPath)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Extracted to %s\n", tempDir)
	return 0
}
