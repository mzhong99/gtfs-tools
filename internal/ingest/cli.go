package ingest

import (
	"fmt"
	"io"
	"errors"
	"flag"
)

type Config struct {
	Version bool

	// Input args - either can accept from zip or url (but not both)
	ZipPath string
	Url string

	// Output args - either can dry-run or write to a database connection
	DryRun bool
	DatabaseConnection string
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
	fs.StringVar(&cfg.ZipPath, "zip", "", "Path to zip file for offline ingest")
	fs.StringVar(&cfg.Url, "url", "", "Path to GTFS URL for online ingest")

	fs.BoolVar(&cfg.DryRun, "dry-run", false, "If specified, shows what would be ingested without performing any DB writes")
	fs.StringVar(&cfg.DatabaseConnection, "database", "", "Path to target database")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	hasZipPath := cfg.ZipPath != "";
	hasUrl := cfg.Url != "";
	if hasZipPath == hasUrl {
		return fmt.Errorf("Exactly one of --zip or --url must be specified.")
	}

	hasDatabaseConnection := cfg.DatabaseConnection != ""
	if hasDatabaseConnection == cfg.DryRun {
		return fmt.Errorf("Exactly one of --dry-run or --database may be specified")
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

	fmt.Printf("+%v\n", cfg)
	return 0
}
