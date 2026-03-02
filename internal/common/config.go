package common

import (
	"context"
	"fmt"

	"github.com/BurntSushi/toml"
	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type RealTimeConfig struct {
	ID          string  `toml:"id"`
	URL         string  `toml:"rt_url"`
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

func (config *RealTimeConfig) ToFeedSpec() (FeedSpec, error) {
	if config.ID == "" {
		return FeedSpec{}, fmt.Errorf("realtime config missing id")
	}
	if config.URL == "" {
		return FeedSpec{}, fmt.Errorf("realtime config %q missing rt_url", config.ID)
	}
	if config.PollSeconds <= 0 {
		return FeedSpec{}, fmt.Errorf("realtime config %q poll_sec must be > 0", config.ID)
	}
	return FeedSpec{
		FeedID:      config.ID,
		URL:         config.URL,
		PollSeconds: config.PollSeconds,
	}, nil
}

func (config *FeedConfig) ToFeedSpecs() ([]FeedSpec, error) {
	feedSpecs := make([]FeedSpec, 0)
	for _, realTimeConfig := range config.RealTime {
		feedSpec, err := realTimeConfig.ToFeedSpec()
		if err != nil {
			return nil, err
		}
		feedSpecs = append(feedSpecs, feedSpec)
	}
	return feedSpecs, nil
}

func (config *SingleConfig) NewDatabase(ctx context.Context) (*database.Database, error) {
	return database.NewDatabaseConnection(ctx, config.Database.URL)
}
