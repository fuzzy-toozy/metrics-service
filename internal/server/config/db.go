package config

import "time"

type DBConfig struct {
	ConnString  string
	DriverName  string
	PingTimeout time.Duration
}
