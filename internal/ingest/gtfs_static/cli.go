package ingest

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"tarediiran-industries.com/gtfs-services/internal/common"
)

type ConfigFile struct {
	DefaultUrl      string `toml:"default_url"`
	DefaultDatabase string `toml:"default_database"`
}

type Config struct {
	Version bool

	// Toml config path - if specified, will ignore all other args and read from this config file instead
	TomlConfigPath string

	// Input args - either can accept from zip or url (but not both)
	ZipPath string
	Url     string

	// Output args - either can dry-run or write to a database connection
	DryRun             *bool
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

	fs.StringVar(&cfg.TomlConfigPath, "toml", "", "Ignore all other args and read from this config file instead")
	fs.StringVar(&cfg.ZipPath, "zip", "", "Path to zip file for offline ingest")
	fs.StringVar(&cfg.Url, "url", "", "Path to GTFS URL for online ingest")

	cfg.DryRun = fs.Bool("dry-run", false, "If specified, shows what would be ingested without performing any DB writes")
	fs.StringVar(&cfg.DatabaseConnection, "database", "", "Path to target database")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Version {
		fmt.Fprintf(errOut, "%s: version %s (%s)\n", programName, common.Version, common.GitCommit)
		return Config{}, flag.ErrHelp
	}

	if cfg.TomlConfigPath != "" {
		tomlCfg, err := LoadConfigFromToml(cfg.TomlConfigPath)
		if err != nil {
			return Config{}, fmt.Errorf("LoadConfigFromToml: %w", err)
		}

		cfg.Url = tomlCfg.DefaultUrl
		cfg.DatabaseConnection = tomlCfg.DefaultDatabase
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	hasZipPath := cfg.ZipPath != ""
	hasUrl := cfg.Url != ""
	if hasZipPath == hasUrl {
		return fmt.Errorf("Exactly one of -zip or -url must be specified.")
	}

	hasDatabaseConnection := cfg.DatabaseConnection != ""
	if hasDatabaseConnection == *cfg.DryRun {
		return fmt.Errorf("Exactly one of -dry-run or -database may be specified")
	}

	return nil
}

func Main(programName string, args []string, stdOut, errOut io.Writer) int {
	cfg, err := ParseArgs(programName, args, errOut)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(errOut, "Error:", err)
		return -1
	}

	return Run(cfg)
}
