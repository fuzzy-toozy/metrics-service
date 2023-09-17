package config

import (
	"flag"
	"os"
	"time"

	"github.com/caarlos0/env"
	"github.com/fuzzy-toozy/metrics-service/internal/config"
)

type Config struct {
	ServerAddress string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
}

func BuildConfig() (*Config, error) {
	var c Config
	defaultTimeout := 30 * time.Second
	readTimeout := config.DurationOption{D: defaultTimeout}
	writeTimeout := config.DurationOption{D: defaultTimeout}
	idleTimeout := config.DurationOption{D: defaultTimeout}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&c.ServerAddress, "a", "localhost:8080", "Address and port to bind server to")
	flag.Var(&readTimeout, "read_timeout", "Server read timeout(seconds)")
	flag.Var(&writeTimeout, "write_timeout", "Server write timeout(seconds)")
	flag.Var(&idleTimeout, "idle_timeout", "Server idle timeout(seconds)")

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	c.ReadTimeout = readTimeout.D
	c.WriteTimeout = writeTimeout.D
	c.IdleTimeout = idleTimeout.D

	err = c.parseEnvVariables()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (config *Config) parseEnvVariables() error {
	type EnvConfig struct {
		ServerAddress string `env:"ADDRESS"`
	}
	ecfg := EnvConfig{}
	err := env.Parse(&ecfg)
	if err != nil {
		return err
	}

	if len(ecfg.ServerAddress) > 0 {
		config.ServerAddress = ecfg.ServerAddress
	}

	return nil
}
