package server

import (
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/fuzzy-toozy/metrics-service/internal/storage"
)

type MetricRegistryHandler struct {
	Registry storage.MetricsStorage
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
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}

	metricType, metricName, metricValue := url[0], url[1], url[2]

	repo, err := h.Registry.GetRepository(metricType)

	if err != nil {
		http.Error(w, "Bad metric type", http.StatusBadRequest)
		return
	}

	err = repo.AddOrUpdate(metricName, metricValue)

	if err != nil {
		http.Error(w, "Bad metric value", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	m, _ := repo.Get(metricName)
	w.Write([]byte(fmt.Sprintf("Metric type '%v', name: '%v', value: '%v' updated. New value: '%v'",
		metricType, metricName, metricValue, m.GetValue())))
}

func hasContentType(r *http.Request, mimetype string) bool {
	contentType := r.Header.Get("Content-type")
	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			break
		}
		if t == mimetype {
			return true
		}
	}
	return false
}
