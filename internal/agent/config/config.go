package config

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type Config struct {
	ServerAddress      string
	ReportURL          string
	ReportBulkURL      string
	ReportEndpoint     string
	ReportBulkEndpoint string
	CompressAlgo       string
	RateLimit          uint
	SecretKey          []byte
	PollInterval       time.Duration
	ReportInterval     time.Duration
}

func getEndpoint(address, url string) string {
	serverEndpoint := address + url
	serverEndpoint = path.Clean(serverEndpoint)
	serverEndpoint = strings.Trim(serverEndpoint, "/")
	serverEndpoint = fmt.Sprintf("http://%v", serverEndpoint)
	return serverEndpoint
}

func (c *Config) Print(log log.Logger) {
	log.Infof("Agent running with config:")
	log.Infof("Server address: %v", c.ServerAddress)
	log.Infof("Report URL: %v", c.ReportURL)
	log.Infof("Report bulk URL: %v", c.ReportBulkURL)
	log.Infof("Report endpoint: %v", c.ReportEndpoint)
	log.Infof("Report bulk endpoint: %v", c.ReportBulkEndpoint)
	log.Infof("Compression algorithm: %v", c.CompressAlgo)
	log.Infof("Rate limit: %v", c.RateLimit)
	log.Infof("Poll interval: %v", c.PollInterval)
	log.Infof("Report interval: %v", c.ReportInterval)
}

func BuildConfig() (*Config, error) {
	c := Config{}
	pollInterval := config.DurationOption{D: 2 * time.Second}
	reportInterval := config.DurationOption{D: 10 * time.Second}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	c.CompressAlgo = "gzip"

	var secretKey string

	flag.StringVar(&secretKey, "k", "", "Secret key")
	flag.StringVar(&c.ServerAddress, "a", "localhost:8080", "Server address")
	flag.StringVar(&c.ReportURL, "u", "/update", "Server endpoint path")
	flag.StringVar(&c.ReportBulkURL, "ub", "/updates", "Server endpoint path")
	flag.UintVar(&c.RateLimit, "l", 20, "Max concurent connections")

	flag.Var(&pollInterval, "p", "Metrics polling interval(seconds)")
	flag.Var(&reportInterval, "r", "Metrics report interval(seconds)")

	c.PollInterval = pollInterval.D
	c.ReportInterval = reportInterval.D

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	if len(secretKey) != 0 {
		c.SecretKey = []byte(secretKey)
	}

	err = c.parseEnvVariables()
	if err != nil {
		return nil, err
	}

	c.ReportEndpoint = getEndpoint(c.ServerAddress, c.ReportURL)
	c.ReportBulkEndpoint = getEndpoint(c.ServerAddress, c.ReportBulkURL)

	return &c, err
}

func (c *Config) parseEnvVariables() error {
	type EnvConfig struct {
		ServerAddress  string `env:"ADDRESS"`
		SecretKey      string `env:"KEY"`
		ReportInterval int    `env:"REPORT_INTERVAL"`
		PollInterval   int    `env:"POLL_INTERVAL"`
		RateLimit      uint   `env:"RATE_LIMIT"`
	}
	ecfg := EnvConfig{}
	err := env.Parse(&ecfg)
	if err != nil {
		return err
	}

	if len(ecfg.SecretKey) > 0 {
		c.SecretKey = []byte(ecfg.SecretKey)
	}

	if len(ecfg.ServerAddress) > 0 {
		c.ServerAddress = ecfg.ServerAddress
	}

	if ecfg.PollInterval > 0 {
		c.PollInterval = time.Duration(ecfg.PollInterval) * time.Second
	}

	if ecfg.ReportInterval > 0 {
		c.PollInterval = time.Duration(ecfg.ReportInterval) * time.Second
	}

	if ecfg.RateLimit > 0 {
		c.RateLimit = ecfg.RateLimit
	}

	return nil
}
