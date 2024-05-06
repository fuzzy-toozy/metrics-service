package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/agent"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	logger := log.NewDevZapLogger()

	logger.Info("Build Info:")
	logger.Infof("    Version: %v", buildVersion)
	logger.Infof("    Date: %v", buildDate)
	logger.Infof("    Commit: %v", buildCommit)

	config, err := config.BuildConfig()
	if err != nil {
		logger.Fatalf("Failed to build agent config: %v", err)
	}

	agent, err := agent.NewAgent(*config, logger, agent.WithCommonMonitor, agent.WithPsMonitor)
	if err != nil {
		logger.Fatalf("Failed to create agent: %v", err)
	}

	agent.Run()
}
