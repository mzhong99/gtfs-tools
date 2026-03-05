package common

import (
	"context"
	"flag"
	"fmt"
	"io"

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
	Feed          FeedConfig          `toml:"feed"`
	Database      DatabaseConfig      `toml:"db"`
	Control       ControlConfig       `toml:"ctl"`
	Observability ObservabilityConfig `toml:"observability"`
}

type ArgsConfig struct {
	Version        bool
	TomlConfigPath string
}

type MainFunction func(cfg SingleConfig) error

func ParseArgs(programName string, args []string, errOut io.Writer) (SingleConfig, error) {
	var argsConfig ArgsConfig

	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errOut)

	fs.Usage = func() {
		fmt.Fprintf(errOut, "Usage: %s [options]\n\n", programName)
		fmt.Fprintln(errOut, "Options")
		fs.PrintDefaults()
	}

	fs.BoolVar(&argsConfig.Version, "version", false, "Prints CLI version")
	fs.StringVar(&argsConfig.TomlConfigPath, "toml", "", "Configuration file")

	if err := fs.Parse(args); err != nil {
		return SingleConfig{}, err
	}

	if argsConfig.Version {
		fmt.Fprintf(errOut, "%s: version %s (%s)\n", programName, Version, GitCommit)
		return SingleConfig{}, flag.ErrHelp
	}

	singleConfig, err := LoadConfigFromToml(argsConfig.TomlConfigPath)
	if err != nil {
		return SingleConfig{}, fmt.Errorf("LoadConfigFromToml: %w", err)
	}
	if err := singleConfig.Validate(); err != nil {
		return SingleConfig{}, err
	}

	return singleConfig, nil
}

func LoadConfigFromToml(path string) (SingleConfig, error) {
	var config SingleConfig
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		return SingleConfig{}, err
	}
	return config, nil
}

func (cfg SingleConfig) Validate() error {
	if len(cfg.Feed.RealTime) == 0 {
		return fmt.Errorf("Need at least one URL to parse real-time feed")
	}
	if cfg.Database.URL == "" {
		return fmt.Errorf("Need path to database to dump real-time feed")
	}
	return nil
}

func ParseArgsAndRun(args []string, out, errOut io.Writer, runner MainFunction) int {
	cfg, err := ParseArgs(args[0], args[1:], errOut)
	if flag.ErrHelp == err {
		return 0
	}
	if err != nil {
		fmt.Fprintln(errOut, "Error:", err)
		return -1
	}

	if err := runner(cfg); err != nil {
		panic(err)
	}

	return 0
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
