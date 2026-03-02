package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type PlatformLayout struct {
	RootDir       string
	ConfigDir     string
	RecordingsDir string
	LogDir        string
	VersionFile   string
}

func NewPlatformLayout(root string) PlatformLayout {
	return PlatformLayout{
		RootDir:       root,
		ConfigDir:     filepath.Join(root, "config"),
		RecordingsDir: filepath.Join(root, "recordings"),
		LogDir:        filepath.Join(root, "logs"),
		VersionFile:   filepath.Join(root, "PLATFORM_VERSION"),
	}
}

func DefaultPlatformRoot() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gtfs-ctl"), nil
}

func InitPlatform(layout PlatformLayout) error {
	dirs := []string{
		layout.RootDir,
		layout.ConfigDir,
		layout.RecordingsDir,
		layout.LogDir,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}

	// Write version file if it doesn't exist
	if _, err := os.Stat(layout.VersionFile); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(layout.VersionFile, []byte("1\n"), 0o644); err != nil {
			return fmt.Errorf("writing version file: %w", err)
		}
	}

	return nil
}
