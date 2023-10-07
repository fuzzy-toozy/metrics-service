package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/storage"
)

type MetricRegistryHandler struct {
	registry storage.MetricsStorage
	log      log.Logger
}

func (h *MetricRegistryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	urlTrimmed := strings.Trim(r.URL.Path, "/")
	url := strings.Split(urlTrimmed, "/")
	if len(url) < 3 {
		w.WriteHeader(http.StatusNotFound)
		h.log.Debugf("Wrong url path %v", r.URL.Path)
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}

	metricType, metricName, metricValue := url[0], url[1], url[2]

	repo, err := h.registry.GetRepository(metricType)

	if err != nil {
		h.log.Debugf("No repository exists for metric: %v. %v", metricType, err)
		http.Error(w, "Bad metric type", http.StatusBadRequest)
		return
	}

	err = repo.AddOrUpdate(metricName, metricValue)

	if err != nil {
		h.log.Debugf("Bad metric value: %v. %v", metricValue, err)
		http.Error(w, "Bad metric value", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	m, _ := repo.Get(metricName)
	logStr := fmt.Sprintf("Metric type '%v', name: '%v', value: '%v' updated. New value: '%v'",
		metricType, metricName, metricValue, m.GetValue())
	h.log.Debugf(logStr)
	w.Write([]byte(logStr))
}

func NewMetricRegistryHandler(registry storage.MetricsStorage, logger log.Logger) *MetricRegistryHandler {
	return &MetricRegistryHandler{registry: registry, log: logger}
}
