// Package config Server database config
package config

import (
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

// DBConfig database configuration.
type DBConfig struct {
	// ConnString database connection string.
	ConnString string
	// DriverName database driver name.
	DriverName string
	// UseDatabase flag indicating to use database or not.
	UseDatabase bool
	// PingTimeout timeout for database operations.
	PingTimeout time.Duration
}

// Print prints database configuration to log.
func (c *DBConfig) Print(logger log.Logger) {
	logger.Infof("Use database: %v", c.UseDatabase)
	logger.Infof("Conn string: %v", c.ConnString)
	logger.Infof("Driver name: %v", c.DriverName)
	logger.Infof("Ping timeout: %v", c.PingTimeout)
}
