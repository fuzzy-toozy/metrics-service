// Package config contains configuration for agent service
package config

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

// Config structure containing various agent service configuration.
type Config struct {
	// ServerAddress address of the metrics server to send metrics to.
	ServerAddress string
	// ReportURL server url to send data to (for single metric).
	ReportURL string
	// ReportBulkURL server url to send data to (for several metrics).
	ReportBulkURL string
	// ReportEndpoint server full endpoint url to send data to including schema (for single metric).
	ReportEndpoint string
	// ReportBulkEndpoint server full endpoint url to send data to including schema (for several metrics).
	ReportBulkEndpoint string
	// CompressAlgo name of compression algorithm to use (only gzip supported atm).
	CompressAlgo string
	// EncKeyPath assymetic encryption public key path.
	EncKeyPath string
	// EncKey assymetic encryption public key.
	EncPublicKey *rsa.PublicKey
	// SecretKey secret key for signing sent data.
	SecretKey []byte
	// PollInterval interval for agent metrics polling.
	PollInterval time.Duration
	// ReportInterval interval for reporting metrics to server.
	ReportInterval time.Duration
	// RateLimit max amount of concurrent connections to server.
	RateLimit uint
}

func getEndpoint(address, url string) string {
	serverEndpoint := address + url
	serverEndpoint = path.Clean(serverEndpoint)
	serverEndpoint = strings.Trim(serverEndpoint, "/")
	serverEndpoint = fmt.Sprintf("http://%v", serverEndpoint)
	return serverEndpoint
}

// Print prints config values to stdout.
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

func parseEncKey(path string) (*rsa.PublicKey, error) {
	encKeyFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open public key file: %v", err)
	}

	encKeyData, err := io.ReadAll(encKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %v", err)
	}

	key, err := encryption.ParseRSAPublicKey(encKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	return key, nil
}

// BuildConfig parses environment varialbes, command line parameters and builds agent's config.
func BuildConfig() (*Config, error) {
	c := Config{}

	const defaultConcurentConnections = 20
	const defaultPollIntervalSec = 2
	const defaultReportIntervalSec = 10

	pollInterval := config.DurationOption{D: defaultPollIntervalSec * time.Second}
	reportInterval := config.DurationOption{D: defaultReportIntervalSec * time.Second}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	c.CompressAlgo = "gzip"

	var secretKey string

	flag.StringVar(&secretKey, "k", "", "Secret key")
	flag.StringVar(&c.EncKeyPath, "crypto-key", "", "Path to public RSA key in PEM format")
	flag.StringVar(&c.ServerAddress, "a", "localhost:8080", "Server address")
	flag.StringVar(&c.ReportURL, "u", "/update", "Server endpoint path")
	flag.StringVar(&c.ReportBulkURL, "ub", "/updates", "Server endpoint path")
	flag.UintVar(&c.RateLimit, "l", defaultConcurentConnections, "Max concurent connections")

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

	if len(c.EncKeyPath) > 0 {
		c.EncPublicKey, err = parseEncKey(c.EncKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %v", err)
		}
	}

	return &c, err
}

func (c *Config) parseEnvVariables() error {
	type EnvConfig struct {
		ServerAddress  string `env:"ADDRESS"`
		SecretKey      string `env:"KEY"`
		EncKeyPath     string `env:"CRYPTO_KEY"`
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

	if len(ecfg.EncKeyPath) > 0 {
		c.EncKeyPath = ecfg.EncKeyPath
	}

	if ecfg.PollInterval > 0 {
		c.PollInterval = time.Duration(ecfg.PollInterval) * time.Second
	}

	if ecfg.ReportInterval > 0 {
		c.ReportInterval = time.Duration(ecfg.ReportInterval) * time.Second
	}

	if ecfg.RateLimit > 0 {
		c.RateLimit = ecfg.RateLimit
	}

	return nil
}
