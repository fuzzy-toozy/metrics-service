package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	// TODO: make application wrapper and move all this code there
	logger := log.NewDevZapLogger()

	logger.Info("Build Info:")
	logger.Infof("    Version: %v", buildVersion)
	logger.Infof("    Date: %v", buildDate)
	logger.Infof("    Commit: %v", buildCommit)

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
