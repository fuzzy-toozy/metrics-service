package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/agent"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

func main() {
	logger := log.NewDevZapLogger()
	config, err := config.BuildConfig()
	if err != nil {
		logger.Fatalf("Failed to build agent config: %v", err)
		return
	}

	metricsStorage := storage.NewCommonMetricsStorage()
	metricsMonitor := monitor.NewMetricsMonitor(metricsStorage, logger)
	psStorage := storage.NewCommonMetricsStorage()
	psMonitor := monitor.NewPsMonitor(psStorage, logger)

	agent := agent.NewAgent(*config, logger, metricsMonitor, psMonitor)
	agent.Run()
}
