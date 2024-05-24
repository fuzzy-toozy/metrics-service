// Package config contains configuration for agent service
package config

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

// Config structure containing various agent service configuration.
type Config struct {
	// ServerAddress address of the metrics server to send metrics to.
	ServerAddress string `json:"address"`
	// ReportURL server url to send data to (for single metric).
	ReportURL string `json:"report_url"`
	// ReportBulkURL server url to send data to (for several metrics).
	ReportBulkURL string `json:"report_bulk_url"`
	// ReportEndpoint server full endpoint url to send data to including schema (for single metric).
	ReportEndpoint string `json:"-"`
	// ReportBulkEndpoint server full endpoint url to send data to including schema (for several metrics).
	ReportBulkEndpoint string `json:"-"`
	// CompressAlgo name of compression algorithm to use (only gzip supported atm).
	CompressAlgo string `json:"compression_algo"`
	// EncKeyPath assymetic encryption public key path.
	EncKeyPath string `json:"crypto_key"`
	// HostIPAddr ip address of current host
	HostIPAddr string `json:"-"`
	// EncKey assymetic encryption public key.
	EncPublicKey *rsa.PublicKey `json:"-"`
	// SecretKey secret key for signing sent data.
	SecretKey []byte `json:"signature_key"`
	// PollInterval interval for agent metrics polling.
	PollInterval config.DurationOption `json:"poll_interval"`
	// ReportInterval interval for reporting metrics to server.
	ReportInterval config.DurationOption `json:"report_interval"`
	// RateLimit max amount of concurrent connections to server.
	RateLimit  uint   `json:"concurrent_connections"`
	ClientType string `json:"client_type"`
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
	log.Infof("Poll interval: %v", c.PollInterval.D)
	log.Infof("Report interval: %v", c.ReportInterval.D)
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

func (c *Config) parseConfigFile(path string) error {
	return config.ParseConfigFile(path, c)
}

func (c *Config) setDefaultValues() {
	const defaultConcurentConnections = 20
	const defaultPollIntervalSec = 2
	const defaultReportIntervalSec = 10
	const defaultReportURL = "/update"
	const defaultReportBulkURL = "/updates"
	const defaultServerAddress = "localhost:8080"
	const defaultCompressAlgo = "gzip"
	const defaultClientType = "http"

	if c.RateLimit == 0 {
		c.RateLimit = defaultConcurentConnections
	}

	if c.PollInterval.D == 0 {
		c.PollInterval.D = defaultPollIntervalSec * time.Second
	}

	if c.ReportInterval.D == 0 {
		c.ReportInterval.D = defaultReportIntervalSec * time.Second
	}

	if len(c.ReportURL) == 0 {
		c.ReportURL = defaultReportURL
	}

	if len(c.ReportBulkURL) == 0 {
		c.ReportBulkURL = defaultReportBulkURL
	}

	if len(c.ServerAddress) == 0 {
		c.ServerAddress = defaultServerAddress
	}

	if len(c.CompressAlgo) == 0 {
		c.CompressAlgo = defaultCompressAlgo
	}

	if len(c.ClientType) == 0 {
		c.ClientType = defaultClientType
	}
}

// BuildConfig parses environment varialbes, command line parameters and builds agent's config.
func BuildConfig() (*Config, error) {
	var (
		secretKey      string
		encKeyPath     string
		serverAddress  string
		reportURL      string
		reportBulkURL  string
		configFilePath string
		clientType     string
		rateLimit      uint
		pollInterval   config.DurationOption
		reportInterval config.DurationOption
	)

	var c Config
	c.setDefaultValues()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&secretKey, "k", "", "Secret key")
	flag.StringVar(&encKeyPath, "crypto-key", "", "Path to public RSA key in PEM format")
	flag.StringVar(&serverAddress, "a", "", "Server address")
	flag.StringVar(&reportURL, "u", "", "Server endpoint path")
	flag.StringVar(&configFilePath, "c", "", "Config file path")
	flag.StringVar(&configFilePath, "config", "", "Config file path")
	flag.StringVar(&clientType, "client", "", "Client type. HTTP or GRPC")

	flag.StringVar(&reportBulkURL, "ub", "", "Server endpoint path")
	flag.UintVar(&rateLimit, "l", 0, "Max concurent connections")

	flag.Var(&pollInterval, "p", "Metrics polling interval")
	flag.Var(&reportInterval, "r", "Metrics report interval")

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	if len(configFilePath) != 0 {
		err = c.parseConfigFile(configFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	if len(secretKey) > 0 {
		c.SecretKey = []byte(secretKey)
	}

	if len(encKeyPath) > 0 {
		c.EncKeyPath = encKeyPath
	}

	if len(serverAddress) > 0 {
		c.ServerAddress = serverAddress
	}

	if len(reportURL) > 0 {
		c.ReportURL = reportURL
	}

	if len(reportBulkURL) > 0 {
		c.ReportBulkURL = reportBulkURL
	}

	if rateLimit > 0 {
		c.RateLimit = rateLimit
	}

	if pollInterval.D > 0 {
		c.PollInterval = pollInterval
	}

	if reportInterval.D > 0 {
		c.ReportInterval = reportInterval
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
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
	}

	addr, err := getOutboundIP(c.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get host ip address: %w", err)
	}

	c.HostIPAddr = addr.String()

	if len(clientType) != 0 {
		c.ClientType = strings.ToLower(clientType)
	}

	if c.ClientType != common.ModeHTTP && c.ClientType != common.ModeGRPC {
		return nil, fmt.Errorf("wrong client type. Only HTTP and GRPC supported")
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
		c.PollInterval.D = time.Duration(ecfg.PollInterval) * time.Second
	}

	if ecfg.ReportInterval > 0 {
		c.ReportInterval.D = time.Duration(ecfg.ReportInterval) * time.Second
	}

	if ecfg.RateLimit > 0 {
		c.RateLimit = ecfg.RateLimit
	}

	return nil
}

func getOutboundIP(serverAddr string) (net.IP, error) {
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Printf("Failed to close connection: %v\n", err)
		}
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
