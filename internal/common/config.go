package common

import (
	"context"

	"github.com/BurntSushi/toml"
	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type RealTimeConfig struct {
	ID          string  `toml:"id"`
	RealTimeURL string  `toml:"rt_url"`
	PollSeconds float64 `toml:"poll_sec"`
}

type FeedConfig struct {
	StaticURL string           `toml:"static_url"`
	RealTime  []RealTimeConfig `toml:"realtime"`
}

type DatabaseConfig struct {
	URL string `toml:"url"`
}

type ControlConfig struct {
	DefaultFormat string `toml:"default_format"`
}

type ObservabilityConfig struct {
	TelemetryUrl string `toml:"telemetry_url"`
}

type SingleConfig struct {
	Feed     FeedConfig     `toml:"feed"`
	Database DatabaseConfig `toml:"db"`
	Control  ControlConfig  `toml:"ctl"`
}

func LoadConfigFromToml(path string) (SingleConfig, error) {
	var config SingleConfig
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return SingleConfig{}, err
	}
	return config, nil
}

func (config *SingleConfig) NewDatabase(ctx context.Context) (*database.Database, error) {
	return database.NewDatabaseConnection(ctx, config.Database.URL)
}
