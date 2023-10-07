package server

import (
	"flag"
	"net/http"
	"time"
)

type Config struct {
	serverAddr string
}

func ParseCmdFlags() *Config {
	var c Config
	flag.StringVar(&c.serverAddr, "a", "localhost:8080", "Address and port to bind server to")
	flag.Parse()
	return &c
}

func NewDefaultHTTPServer() *http.Server {
	c := ParseCmdFlags()
	h := NewDefaultMetricRegistryHandler()

	s := http.Server{
		Addr:         c.serverAddr,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      SetupRouting(h),
	}

	return &s
}
