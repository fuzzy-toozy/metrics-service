package main

import (
	"net/http"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/server"
	"github.com/fuzzy-toozy/metrics-service/internal/storage"
)

func main() {
	h := server.MetricRegistryHandler{
		Registry: storage.NewCommonMetricsStorage(),
	}

	h.Registry.AddRepository("gauge", storage.NewGaugeMetricRepository())
	h.Registry.AddRepository("counter", storage.NewCounterMetricRepository())

	mux := http.NewServeMux()
	mux.Handle("/update/", http.StripPrefix("/update/", &h))
	s := http.Server{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      mux,
	}

	err := s.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}
