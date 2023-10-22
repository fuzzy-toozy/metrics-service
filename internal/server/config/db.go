package config

import "time"

type DBConfig struct {
	UseDatabase bool
	ConnString  string
	DriverName  string
	PingTimeout time.Duration
}
