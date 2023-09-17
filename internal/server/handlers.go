package server

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/storage"
	"github.com/go-chi/chi"
)

type MetricURLInfo struct {
	Type  string
	Name  string
	Value string
}

type MetricRegistryHandler struct {
	registry   storage.MetricsStorage
	log        log.Logger
	metricInfo MetricURLInfo
	allMetrics *template.Template
}

func (h *MetricRegistryHandler) GetMetricURLInfo() MetricURLInfo {
	return h.metricInfo
}

func (h *MetricRegistryHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))
	metricName := strings.ToLower(chi.URLParam(r, h.metricInfo.Name))

	repo, err := h.registry.GetRepository(metricType)

	if err != nil {
		h.log.Debugf("No repository exists for metric: %v. %v", metricType, err)
		http.Error(w, "Bad metric type", http.StatusBadRequest)
		return
	}

	m, err := repo.Get(metricName)

	if err != nil {
		h.log.Debugf("Metric get failed: %v. %v", metricName, err)
		http.Error(w, "Metric not fould", http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(m.GetValue()))
}

func (h *MetricRegistryHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	type MetricInfo struct {
		Name string
		Val  string
	}

	metrics := []MetricInfo{}

	h.registry.ForEachRepository(func(name string, r storage.Repository) error {
		return r.ForEachMetric(func(name string, m storage.Metric) error {
			metrics = append(metrics, MetricInfo{Name: name, Val: m.GetValue()})
			return nil
		})
	})

	h.log.Infof("METRICS len %v", len(metrics))

	pageTempl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>All metics</title>
</head>
<body>
<table>
  <tr>
    <th>Name</th>
    <th>Value</th>
  </tr>
{{range .}}
<tr>
<th>{{.Name}}</th>
<th>{{.Val}}</th>
</tr>
{{end}}
</table>
</body>
</html>`

	if h.allMetrics == nil {
		tmpl, err := template.New("AllMetrics").Parse(pageTempl)

		if err != nil {
			h.log.Debugf("Parsing template failed: %v", err)
			http.Error(w, "Bad metric value", http.StatusBadRequest)
			return
		}

		h.allMetrics = tmpl
	}

	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	h.allMetrics.Execute(w, metrics)
}

func (h *MetricRegistryHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))
	metricName := strings.ToLower(chi.URLParam(r, h.metricInfo.Name))
	metricValue := strings.ToLower(chi.URLParam(r, h.metricInfo.Value))

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

func NewMetricRegistryHandler(registry storage.MetricsStorage, logger log.Logger, minfo MetricURLInfo) *MetricRegistryHandler {
	return &MetricRegistryHandler{registry: registry, log: logger, metricInfo: minfo}
}

func NewDefaultMetricRegistryHandler() *MetricRegistryHandler {
	registry := storage.NewCommonMetricsStorage()
	registry.AddRepository("gauge", storage.NewGaugeMetricRepository())
	registry.AddRepository("counter", storage.NewCounterMetricRepository())

	minfo := MetricURLInfo{
		Name:  "metricName",
		Value: "metricValue",
		Type:  "metricType",
	}

	return NewMetricRegistryHandler(registry, log.NewDevZapLogger(), minfo)
}

func SetupRouting(h *MetricRegistryHandler) http.Handler {
	r := chi.NewRouter()
	minfo := h.GetMetricURLInfo()
	r.Route("/update", func(r chi.Router) {
		r.Post(fmt.Sprintf("/{%v}/{%v}/{%v}", minfo.Type, minfo.Name, minfo.Value),
			func(w http.ResponseWriter, r *http.Request) {
				h.UpdateMetric(w, r)
			})
	})

	r.Route("/value", func(r chi.Router) {
		r.Get(fmt.Sprintf("/{%v}/{%v}", minfo.Type, minfo.Name), func(w http.ResponseWriter, r *http.Request) {
			h.GetMetric(w, r)
		})
	})

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			h.GetAllMetrics(w, r)
		})
	})

	return r
}
