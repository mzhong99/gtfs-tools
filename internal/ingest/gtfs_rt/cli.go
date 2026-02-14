package gtfs_rt

import (
	"flag"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"tarediiran-industries.com/gtfs-services/internal/common"
)

type ConfigFile struct {
	Urls               []string `toml:"urls"`
	DatabaseConnection string   `toml:"database"`
}

type Config struct {
	Version            bool
	TomlConfigPath     string
	Urls               []string
	DatabaseConnection string
}

func LoadConfigFromToml(path string) (ConfigFile, error) {
	var cfg ConfigFile
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return ConfigFile{}, err
	}

	return cfg, nil
}

func ParseArgs(programName string, args []string, errOut io.Writer) (Config, error) {
	var cfg Config

	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errOut)

	fs.Usage = func() {
		fmt.Fprintf(errOut, "Usage: %s [options]\n\n", programName)
		fmt.Fprintln(errOut, "Options")
		fs.PrintDefaults()
	}

	fs.BoolVar(&cfg.Version, "version", false, "Prints CLI version")
	fs.StringVar(&cfg.TomlConfigPath, "toml", "", "Configuration file")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Version {
		fmt.Fprintf(errOut, "%s: version %s (%s)\n", programName, common.Version, common.GitCommit)
		return cfg, flag.ErrHelp
	}

	if cfg.TomlConfigPath != "" {
		tomlCfg, err := LoadConfigFromToml(cfg.TomlConfigPath)
		if err != nil {
			return Config{}, fmt.Errorf("LoadConfigFromToml: %w", err)
		}

		cfg.Urls = tomlCfg.Urls
		cfg.DatabaseConnection = tomlCfg.DatabaseConnection
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if len(cfg.Urls) == 0 {
		return fmt.Errorf("Need at least one URL to parse real-time feed")
	}
	if cfg.DatabaseConnection == "" {
		return fmt.Errorf("Missing required argument: database")
	}
	return nil
}

func Main(programName string, args []string, out, errOut io.Writer) int {
	cfg, err := ParseArgs(programName, args, errOut)
	if err != nil {
		if flag.ErrHelp == err {
			return 0
		}
		fmt.Fprintln(errOut, "Error:", err)
		return -1
	}

	return Run(cfg)
}
