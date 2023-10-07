package main

import (
	"net/http"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server"
	"github.com/fuzzy-toozy/metrics-service/internal/storage"
)

func main() {
	registry := storage.NewCommonMetricsStorage()
	registry.AddRepository("gauge", storage.NewGaugeMetricRepository())
	registry.AddRepository("counter", storage.NewCounterMetricRepository())

	minfo := server.MetricURLInfo{
		Name:  "metricName",
		Value: "metricValue",
		Type:  "metricType",
	}

	h := server.NewMetricRegistryHandler(registry, log.NewDevZapLogger(), minfo)

	s := http.Server{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      server.SetupRouting(h),
	}

	err := s.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}
