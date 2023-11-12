package config

import (
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type DBConfig struct {
	UseDatabase bool
	ConnString  string
	DriverName  string
	PingTimeout time.Duration
}

func (c *DBConfig) Print(logger log.Logger) {
	logger.Infof("Use database: %v", c.UseDatabase)
	logger.Infof("Conn string: %v", c.ConnString)
	logger.Infof("Driver name: %v", c.DriverName)
	logger.Infof("Ping timeout: %v", c.PingTimeout)
}
