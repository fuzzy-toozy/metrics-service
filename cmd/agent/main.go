package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/agent"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/http"
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

	client := http.NewDefaultHTTPClient()
	metricsStorage := storage.NewCommonMetricsStorage()
	metricsMonitor := monitor.NewCommonMonitor(metricsStorage, logger)

	agent := agent.NewAgent(*config, client, metricsMonitor, logger)
	agent.Run()
}
