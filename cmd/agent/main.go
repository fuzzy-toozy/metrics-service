package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/agent"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

func main() {
	logger := log.NewDevZapLogger()
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
