package config

import (
	"flag"
	"os"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
)

type Config struct {
	ServerAddress  string
	ReportURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func BuildConfig() (*Config, error) {
	c := Config{}
	pollInterval := config.DurationOption{D: 2 * time.Second}
	reportInterval := config.DurationOption{D: 10 * time.Second}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	flag.StringVar(&c.ServerAddress, "a", "localhost:8080", "Server address")
	flag.StringVar(&c.ReportURL, "u", "/update", "Server endpoint path")
	flag.Var(&pollInterval, "p", "Metrics polling interval(seconds)")
	flag.Var(&reportInterval, "r", "Metrics report interval(seconds)")

	c.PollInterval = pollInterval.D
	c.ReportInterval = reportInterval.D

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	err = c.parseEnvVariables()
	if err != nil {
		return nil, err
	}

	return &c, err
}

func (config *Config) parseEnvVariables() error {
	type EnvConfig struct {
		ServerAddress  string `env:"ADDRESS"`
		ReportInterval int    `env:"REPORT_INTERVAL"`
		PollInterval   int    `env:"POLL_INTERVAL"`
	}
	ecfg := EnvConfig{}
	err := env.Parse(&ecfg)
	if err != nil {
		return err
	}

	if len(ecfg.ServerAddress) > 0 {
		config.ServerAddress = ecfg.ServerAddress
	}

	if ecfg.PollInterval > 0 {
		config.PollInterval = time.Duration(ecfg.PollInterval) * time.Second
	}

	if ecfg.ReportInterval > 0 {
		config.PollInterval = time.Duration(ecfg.ReportInterval) * time.Second
	}

	return nil
}
