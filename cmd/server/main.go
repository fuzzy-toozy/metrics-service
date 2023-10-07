package main

import (
	"context"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/fuzzy-toozy/metrics-service/internal/server/routing"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
)

func main() {
	logger := log.NewDevZapLogger()
	config, err := config.BuildConfig()
	if err != nil {
		logger.Fatalf("Failed to build server config: %v", err)
		return
	}
	metricsStorage := storage.NewCommonMetricsStorage()
	registryHandler := handlers.NewDefaultMetricRegistryHandler(logger, metricsStorage)
	routerHandler := routing.SetupRouting(registryHandler)
	loggingHandler := handlers.WithLogging(routerHandler, logger)

	s := server.NewDefaultHTTPServer(*config, logger, loggingHandler)

	err = server.Run(
		func() error {
			return s.ListenAndServe()
		},
		func() error {
			err := s.Shutdown(context.Background())
			if err != nil {
				logger.Error("Shutdown failed. Reason: %v", err)
			}
			return err
		},
	)

	if err != nil {
		logger.Warnf("Stop reason: %v", err)
	}
}
