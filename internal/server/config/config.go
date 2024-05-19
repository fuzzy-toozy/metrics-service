// Package config Server configuration.
package config

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type Config struct {
	// ServerAddress address of the server eg localhost:8080.
	ServerAddress string `json:"address"`
	// StoreFilePath path to store metrics storage backup (in case no database used).
	StoreFilePath string `json:"store_file"`
	// Assymetric encryption private key path
	EncKeyPath string `json:"crypto_key"`
	// DbConnString database connection string.
	DBConnString  string `json:"database_dsn"`
	TrustedSubnet string `json:"trusted_subnet"`
	// TrustedSubnetAddr Parsed subnet to accept requests from
	TrustedSubnetAddr *net.IPNet `json:"-"`
	// SecretKey key to validate signature of sent data.
	SecretKey []byte `json:"signature_key"`
	// Assymetric encryption private key
	EncryptPrivKey *rsa.PrivateKey `json:"-"`
	// DatabaseConfig database configuration.
	DatabaseConfig DBConfig `json:"-"`
	// StoreInterval interval between storage backups (in case no database used).
	StoreInterval config.DurationOption `json:"store_interval"`
	// RestoreData instructs to attempt to restore data from file backupt (in case no database used).
	RestoreData bool `json:"restore"`
	// MaxBodySize max size of http request body.
	MaxBodySize uint64 `json:"max_body_size"`
	// ReadTimeout timeout for reading request data from client.
	ReadTimeout config.DurationOption `json:"read_timeout"`
	// WriteTimeout timeout for writing response data to client.
	WriteTimeout config.DurationOption `json:"write_timeout"`
	// IdleTimeout maximum duration for which the server should keep an idle connection open before closing it.
	IdleTimeout config.DurationOption `json:"idle_timeout"`
	// TrustedSubnet Subnet in CIDR format to accept requests from
}

// Print writes server configuration to log.
func (c *Config) Print(logger log.Logger) {
	logger.Infof("Server running with config:")
	logger.Infof("Server address: %v", c.ServerAddress)
	logger.Infof("Store file path: %v", c.StoreFilePath)
	logger.Infof("Store interval: %v", c.StoreInterval.D)
	logger.Infof("Restore data: %v", c.RestoreData)
	logger.Infof("Max request body size: %v", c.MaxBodySize)
	logger.Infof("Read timeout: %v", c.ReadTimeout.D)
	logger.Infof("Write timeout: %v", c.WriteTimeout.D)
	logger.Infof("Idle timeout: %v", c.IdleTimeout.D)

	logger.Infof("Database config:")
	c.DatabaseConfig.Print(logger)
}

func (c *Config) parseConfigFile(path string) error {
	return config.ParseConfigFile(path, c)
}

func parseEncKey(path string) (*rsa.PrivateKey, error) {
	encKeyFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open private key file: %v", err)
	}

	encKeyData, err := io.ReadAll(encKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}

	key, err := encryption.ParseRSAPrivateKey(encKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return key, nil
}

func (c *Config) setDefaultValues() {
	const (
		defaultMaxBodySize   = 1048576
		defaultPingTimeout   = 2
		defaultCommonTimeout = 30
		defaultStoreInterval = 300
		defaultServerAddress = "localhost:8080"
		defaultStoreFilePath = "/tmp/metrics-db.json"
		defaultDBDriver      = "pgx"
	)

	if c.MaxBodySize == 0 {
		c.MaxBodySize = defaultMaxBodySize
	}

	if len(c.StoreFilePath) == 0 {
		c.StoreFilePath = defaultStoreFilePath
	}

	if len(c.ServerAddress) == 0 {
		c.ServerAddress = defaultServerAddress
	}

	if c.StoreInterval.D == 0 {
		c.StoreInterval.D = defaultStoreInterval * time.Second
	}

	if len(c.DatabaseConfig.DriverName) == 0 {
		c.DatabaseConfig.DriverName = defaultDBDriver
	}

	if c.DatabaseConfig.PingTimeout == 0 {
		c.DatabaseConfig.PingTimeout = defaultPingTimeout * time.Second
	}

	if c.ReadTimeout.D == 0 {
		c.ReadTimeout.D = defaultCommonTimeout * time.Second
	}

	if c.WriteTimeout.D == 0 {
		c.WriteTimeout.D = defaultCommonTimeout * time.Second
	}

	if c.IdleTimeout.D == 0 {
		c.IdleTimeout.D = defaultCommonTimeout * time.Second
	}
}

// BuildConfig parses command line parameters and environment variables
// and builds configuration from parsed data.
func BuildConfig() (*Config, error) {
	var (
		dbConnString   string
		encKeyPath     string
		secretKey      string
		serverAddress  string
		storeFilePath  string
		configFilePath string
		maxBodySize    uint64
		pingTimeout    config.DurationOption
		readTimeout    config.DurationOption
		writeTimeout   config.DurationOption
		idleTimeout    config.DurationOption
		storeInterval  config.DurationOption
	)

	var c Config
	c.setDefaultValues()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&secretKey, "k", "", "Sever secret key")
	flag.StringVar(&dbConnString, "d", "", "Database connection string")
	flag.StringVar(&serverAddress, "a", "", "Address and port to bind server to")
	flag.StringVar(&storeFilePath, "f", "", "File to store metrics data to")
	flag.StringVar(&encKeyPath, "crypto-key", "", "Path to private RSA key in PEM format")
	flag.BoolVar(&c.RestoreData, "r", true, "Restore data from previously stored values")
	flag.Uint64Var(&maxBodySize, "bs", 0, "Max HTTP body size")
	flag.StringVar(&configFilePath, "c", "", "Config file path")
	flag.StringVar(&configFilePath, "config", "", "Config file path")
	flag.StringVar(&c.TrustedSubnet, "t", "", "Subnet to accept requests from")

	flag.Var(&pingTimeout, "ping_timeout", "DB ping timeout and retry timeout")
	flag.Var(&readTimeout, "read_timeout", "Server read timeout(seconds)")
	flag.Var(&writeTimeout, "write_timeout", "Server write timeout(seconds)")
	flag.Var(&idleTimeout, "idle_timeout", "Server idle timeout(seconds)")
	flag.Var(&storeInterval, "i", "Save data to NVM interval")

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	if len(configFilePath) != 0 {
		err = c.parseConfigFile(configFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		c.DatabaseConfig.ConnString = c.DBConnString
	}

	if len(secretKey) > 0 {
		c.SecretKey = []byte(secretKey)
	}

	if len(dbConnString) > 0 {
		c.DatabaseConfig.ConnString = dbConnString
	}

	if len(serverAddress) > 0 {
		c.ServerAddress = serverAddress
	}

	if len(storeFilePath) > 0 {
		c.StoreFilePath = storeFilePath
	}

	if len(encKeyPath) > 0 {
		c.EncKeyPath = encKeyPath
	}

	if maxBodySize > 0 {
		c.MaxBodySize = maxBodySize
	}

	if pingTimeout.D > 0 {
		c.DatabaseConfig.PingTimeout = pingTimeout.D
	}

	if idleTimeout.D > 0 {
		c.IdleTimeout = idleTimeout
	}

	if readTimeout.D > 0 {
		c.ReadTimeout = readTimeout
	}

	if writeTimeout.D > 0 {
		c.WriteTimeout = writeTimeout
	}

	if storeInterval.D > 0 {
		c.StoreInterval = storeInterval
	}

	err = c.ParseEnvVariables()
	if err != nil {
		return nil, err
	}

	if len(c.DatabaseConfig.ConnString) > 0 {
		c.DatabaseConfig.UseDatabase = true
	}

	if len(c.EncKeyPath) > 0 {
		c.EncryptPrivKey, err = parseEncKey(c.EncKeyPath)
		if err != nil {
			return nil, err
		}
	}
	if len(c.TrustedSubnet) > 0 {
		_, c.TrustedSubnetAddr, err = net.ParseCIDR(c.TrustedSubnet)
		if err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// ParseEnvVariables parses environment variables
// and builds configuration from parsed data.
func (c *Config) ParseEnvVariables() error {
	type EnvConfig struct {
		ServerAddress string `env:"ADDRESS"`
		StoreInterval string `env:"STORE_INTERVAL"`
		StoragePath   string `env:"FILE_STORAGE_PATH"`
		Restore       string `env:"RESTORE"`
		DBConnStr     string `env:"DATABASE_DSN"`
		SecretKey     string `env:"KEY"`
		EncKeyPath    string `env:"CRYPTO_KEY"`
		TrustedSubnet string `env:"TRUSTED_SUBNET"`
	}
	ecfg := EnvConfig{}
	err := env.Parse(&ecfg)
	if err != nil {
		return err
	}

	if len(ecfg.ServerAddress) > 0 {
		c.ServerAddress = ecfg.ServerAddress
	}

	if len(ecfg.StoreInterval) > 0 {
		val, err := strconv.ParseUint(ecfg.StoreInterval, 10, 64)
		if err != nil {
			return err
		}
		c.StoreInterval.D = time.Duration(val * uint64(time.Second))
	}

	if len(ecfg.StoragePath) > 0 {
		c.StoreFilePath = ecfg.StoragePath
	}

	if len(ecfg.Restore) > 0 {
		val, err := strconv.ParseBool(ecfg.Restore)
		if err != nil {
			return err
		}
		c.RestoreData = val
	}

	if len(ecfg.DBConnStr) > 0 {
		c.DatabaseConfig.ConnString = ecfg.DBConnStr
	}

	if len(ecfg.SecretKey) > 0 {
		c.SecretKey = []byte(ecfg.SecretKey)
	}

	if len(ecfg.TrustedSubnet) > 0 {
		c.TrustedSubnet = ecfg.TrustedSubnet
	}

	return nil
}
