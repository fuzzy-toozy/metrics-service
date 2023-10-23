package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server"
)

func main() {
	// TODO: make application wrapper and move all this code there
	logger := log.NewDevZapLogger()
	server, err := server.NewServer(logger)

	if err != nil {
		logger.Errorf("Failed to create server: %v", err)
		return
	}

	err = server.Run()

	if err != nil {
		logger.Warnf("Stop reason: %v", err)
	}
}
