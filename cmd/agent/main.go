package main

import (
	"github.com/fuzzy-toozy/metrics-service/internal/agent"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/monitor"
)

func main() {
	logger := log.NewDevZapLogger()
	agent := agent.NewAgent(*agent.ParseCmdFlags(), agent.NewDefaultHTTPClient(),
		monitor.NewCommonMonitor(monitor.NewCommonMetricsStorage(), logger), logger)
	agent.Run()
}
