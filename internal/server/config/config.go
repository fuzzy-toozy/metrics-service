// Package config Server configuration.
package config

import (
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

const defaultMaxBodySize = 1048576
const defaultPingTimeout = 2
const defaultCommonTimeout = 30
const defaultStoreInterval = 300

type Config struct {
	// ServerAddress address of the server eg localhost:8080.
	ServerAddress string
	// StoreFilePath path to store metrics storage backup (in case no database used).
	StoreFilePath string
	// Assymetric encryption private key path
	EncKeyPath string
	// SecretKey key to validate signature of sent data.
	SecretKey []byte
	// Assymetric encryption private key
	EncryptPrivKey *rsa.PrivateKey
	// DatabaseConfig database configuration.
	DatabaseConfig DBConfig
	// StoreInterval interval between storage backups (in case no database used).
	StoreInterval time.Duration
	// RestoreData instructs to attempt to restore data from file backupt (in case no database used).
	RestoreData bool
	// MaxBodySize max size of http request body.
	MaxBodySize uint64
	// ReadTimeout timeout for reading request data from client.
	ReadTimeout time.Duration
	// WriteTimeout timeout for writing response data to client.
	WriteTimeout time.Duration
	// IdleTimeout maximum duration for which the server should keep an idle connection open before closing it.
	IdleTimeout time.Duration
}

// Print writes server configuration to log.
func (c *Config) Print(logger log.Logger) {
	logger.Infof("Server running with config:")
	logger.Infof("Server address: %v", c.ServerAddress)
	logger.Infof("Store file path: %v", c.ServerAddress)
	logger.Infof("Store interval: %v", c.IdleTimeout)
	logger.Infof("Restore data: %v", c.RestoreData)
	logger.Infof("Max request body size: %v", c.MaxBodySize)
	logger.Infof("Read timeout: %v", c.ReadTimeout)
	logger.Infof("Write timeout: %v", c.WriteTimeout)
	logger.Infof("Idle timeout: %v", c.IdleTimeout)

	logger.Infof("Database config:")
	c.DatabaseConfig.Print(logger)
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

// BuildConfig parses command line parameters and environment variables
// and builds configuration from parsed data.
func BuildConfig() (*Config, error) {
	var c Config
	c.MaxBodySize = defaultMaxBodySize
	c.DatabaseConfig.DriverName = "pgx"
	c.DatabaseConfig.PingTimeout = defaultPingTimeout * time.Second
	var secretKey string

	defaultTimeout := defaultCommonTimeout * time.Second
	readTimeout := config.DurationOption{D: defaultTimeout}
	writeTimeout := config.DurationOption{D: defaultTimeout}
	idleTimeout := config.DurationOption{D: defaultTimeout}
	storeInterval := config.DurationOption{D: defaultStoreInterval * time.Second}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&secretKey, "k", "", "Sever secret key")
	flag.StringVar(&c.DatabaseConfig.ConnString, "d", "",
		"Database connection string")
	flag.StringVar(&c.ServerAddress, "a", "localhost:8080", "Address and port to bind server to")
	flag.StringVar(&c.StoreFilePath, "f", "/tmp/metrics-db.json", "File to store metrics data to")
	flag.StringVar(&c.EncKeyPath, "crypto-key", "", "Path to private RSA key in PEM format")
	flag.BoolVar(&c.RestoreData, "r", true, "Restore data from previously stored values")

	flag.Var(&readTimeout, "read_timeout", "Server read timeout(seconds)")
	flag.Var(&writeTimeout, "write_timeout", "Server write timeout(seconds)")
	flag.Var(&idleTimeout, "idle_timeout", "Server idle timeout(seconds)")
	flag.Var(&storeInterval, "i", "Save data to NVM interval")

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	if len(secretKey) > 0 {
		c.SecretKey = []byte(secretKey)
	}

	c.ReadTimeout = readTimeout.D
	c.WriteTimeout = writeTimeout.D
	c.IdleTimeout = idleTimeout.D
	c.StoreInterval = storeInterval.D

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
		c.StoreInterval = time.Duration(val * uint64(time.Second))
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

	return nil
}
