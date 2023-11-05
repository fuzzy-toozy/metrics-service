package config

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
)

type Config struct {
	ServerAddress  string
	StoreFilePath  string
	SecretKey      []byte
	DatabaseConfig DBConfig
	RestoreData    bool
	MaxBodySize    uint64
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	StoreInterval  time.Duration
}

func BuildConfig() (*Config, error) {
	var c Config
	c.MaxBodySize = 1024 * 1024
	c.DatabaseConfig.DriverName = "pgx"
	c.DatabaseConfig.PingTimeout = 2 * time.Second
	var secretKey string

	defaultTimeout := 30 * time.Second
	readTimeout := config.DurationOption{D: defaultTimeout}
	writeTimeout := config.DurationOption{D: defaultTimeout}
	idleTimeout := config.DurationOption{D: defaultTimeout}
	storeInterval := config.DurationOption{D: 300 * time.Second}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&secretKey, "k", "", "Sever secret key")
	flag.StringVar(&c.DatabaseConfig.ConnString, "d", "",
		"Database connection string")
	flag.StringVar(&c.ServerAddress, "a", "localhost:8080", "Address and port to bind server to")
	flag.StringVar(&c.StoreFilePath, "f", "/tmp/metrics-db.json", "File to store metrics data to")
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

	err = c.parseEnvVariables()
	if err != nil {
		return nil, err
	}

	if len(c.DatabaseConfig.ConnString) > 0 {
		c.DatabaseConfig.UseDatabase = true
	}

	return &c, nil
}

func (config *Config) parseEnvVariables() error {
	type EnvConfig struct {
		ServerAddress string `env:"ADDRESS"`
		StoreInterval string `env:"STORE_INTERVAL"`
		StoragePath   string `env:"FILE_STORAGE_PATH"`
		Restore       string `env:"RESTORE"`
		DBConnStr     string `env:"DATABASE_DSN"`
		SecretKey     string `env:"KEY"`
	}
	ecfg := EnvConfig{}
	err := env.Parse(&ecfg)
	if err != nil {
		return err
	}

	if len(ecfg.ServerAddress) > 0 {
		config.ServerAddress = ecfg.ServerAddress
	}

	if len(ecfg.StoreInterval) > 0 {
		val, err := strconv.ParseUint(ecfg.StoreInterval, 10, 64)
		if err != nil {
			return err
		}
		config.StoreInterval = time.Duration(val * uint64(time.Second))
	}

	if len(ecfg.StoragePath) > 0 {
		config.StoreFilePath = ecfg.StoragePath
	}

	if len(ecfg.Restore) > 0 {
		val, err := strconv.ParseBool(ecfg.Restore)
		if err != nil {
			return err
		}
		config.RestoreData = val
	}

	if len(ecfg.DBConnStr) > 0 {
		config.DatabaseConfig.ConnString = ecfg.DBConnStr
	}

	if len(ecfg.SecretKey) > 0 {
		config.SecretKey = []byte(ecfg.SecretKey)
	}

	return nil
}
