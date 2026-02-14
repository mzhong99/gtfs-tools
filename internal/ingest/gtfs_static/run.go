package gtfs_static

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
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

func UnzipToTempDir(zipPath string) (string, error) {
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

		_, err = io.Copy(dstFile, fileInArchive)
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("Extracted %s -> %s\n", zipPath, dir)

	return dir, nil
}

func DownloadToTempFile(url string) (string, error) {
	tmpFile, error := os.CreateTemp("", "gtfs-ingest-*.zip")
	if error != nil {
		return "", error
	}
	defer tmpFile.Close()

	response, err := http.Get(url)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("Failed to download %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("HTTP Error: status code %d", response.StatusCode)
	}

	_, err = io.Copy(tmpFile, response.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("Failed to write downloaded file to temp location: %w", err)
	}

	fmt.Printf("Downloaded %s -> %s\n", url, tmpFile.Name())
	return tmpFile.Name(), nil
}

func Run(cfg Config) int {
	zipPath := ""
	if cfg.Url != "" {
		var err error
		zipPath, err = DownloadToTempFile(cfg.Url)
		if err != nil {
			panic(err)
		}
	} else {
		zipPath = cfg.ZipPath
	}

	if err := LoadGtfsFromDirectory(cfg.Url, zipPath, cfg.DatabaseConnection); err != nil {
		panic(err)
	}

	return 0
}
