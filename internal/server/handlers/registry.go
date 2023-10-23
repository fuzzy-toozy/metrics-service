package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"text/template"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/errtypes"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"github.com/go-chi/chi"
)

type MetricURLInfo struct {
	Type  string
	Name  string
	Value string
}

type MetricRegistryHandler struct {
	registry       storage.Repository
	log            log.Logger
	metricInfo     MetricURLInfo
	allMetrics     *template.Template
	storageSaver   storage.StorageSaver
	databaseConfig config.DBConfig
}

func (h *MetricRegistryHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	err := h.registry.HealthCheck()

	if err != nil {
		h.log.Errorf("Failed to perform registry health check: %v", err)
		http.Error(w, "Ping failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *MetricRegistryHandler) GetMetricURLInfo() MetricURLInfo {
	return h.metricInfo
}

func (h *MetricRegistryHandler) getMetric(name string, mtype string) (val string, status int, err error) {
	m, err := h.registry.Get(name, mtype)

	status = errtypes.ErrorToStatus(err)

	if err != nil {
		return "", status, err
	}

	val, err = m.GetData()
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	return val, http.StatusOK, nil
}

func (h *MetricRegistryHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))

	metricName := chi.URLParam(r, h.metricInfo.Name)

	val, status, err := h.getMetric(metricName, metricType)

	if err != nil {
		h.log.Debugf("failed to get metric: %v", err)
	}

	w.WriteHeader(status)
	w.Write([]byte(val))
}

func (h *MetricRegistryHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := metrics.Metric{}

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to decode JSON data: %v", err)
		w.Write([]byte("Bad metric format"))
		return
	}

	value, status, err := h.getMetric(receivedData.ID, receivedData.MType)

	if err != nil {
		h.log.Errorf("Failed to get metric of type %v, name %v: %v", receivedData.MType, receivedData.ID, err)
	}

	if status != http.StatusOK {
		w.WriteHeader(status)
		w.Write([]byte(value))
		return
	}

	respData, _ := metrics.NewMetric(receivedData.ID, value, receivedData.MType)

	if err := receivedData.UpdateData(value); err != nil {
		h.log.Debugf("Failed to get metric %v", err)
		http.Error(w, "Failed to get metric", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(respData)
}

func (h *MetricRegistryHandler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {

	type MetricInfo struct {
		Name string
		Val  string
	}

	repoMetrics, err := h.registry.GetAll()

	if err != nil {
		h.log.Errorf("Failed to get all metrics: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	metrics := make([]MetricInfo, 0, len(repoMetrics))
	for _, m := range repoMetrics {
		data, err := m.GetData()
		if err != nil {
			h.log.Errorf("Failed to get all metrics: %v", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		metrics = append(metrics, MetricInfo{Name: m.ID, Val: data})
	}

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

func (h *MetricRegistryHandler) updateMetric(mtype, mname, mvalue string) (metricValue string, statusCode int, err error) {
	updatedVal, err := h.registry.AddOrUpdate(mname, mvalue, mtype)

	status := errtypes.ErrorToStatus(err)
	if err != nil {
		return "", status, err
	}

	if h.storageSaver != nil {
		err := h.storageSaver.Save()
		if err != nil {
			h.log.Errorf("Failed to update persistent storage: %v", err)
		}
	}

	return updatedVal, status, nil
}

func (h *MetricRegistryHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))
	metricValue := strings.ToLower(chi.URLParam(r, h.metricInfo.Value))
	metricName := chi.URLParam(r, h.metricInfo.Name)

	value, status, err := h.updateMetric(metricType, metricName, metricValue)

	if err != nil {
		h.log.Debugf("Failed to update metric: %v", err)
	}

	w.WriteHeader(status)
	w.Write([]byte(value))
}

func (h *MetricRegistryHandler) UpdateMetricsFromJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := make([]metrics.Metric, 0)

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to decode JSON data: %v", err)
		w.Write([]byte("Bad metric format"))
		return
	}

	err := h.registry.AddMetricsBulk(receivedData)
	status := errtypes.ErrorToStatus(err)
	if err != nil {
		h.log.Errorf("Failed to add metrics: %v", err)
		http.Error(w, "", status)
	}

	respMetrics, err := h.registry.GetAll()
	status = errtypes.ErrorToStatus(err)
	if err != nil {
		h.log.Errorf("Failed to add metrics: %v", err)
		http.Error(w, "", status)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(respMetrics)
}

func (h *MetricRegistryHandler) UpdateMetricFromJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := metrics.Metric{}

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to decode JSON data: %v", err)
		w.Write([]byte("Bad metric format"))
		return
	}

	value, err := receivedData.GetData()
	if err != nil {
		h.log.Debugf("Failed to get metric data: %v", err)
		http.Error(w, "Bad metric data in request", http.StatusBadRequest)
		return
	}

	value, status, err := h.updateMetric(receivedData.MType, receivedData.ID, value)

	respData, _ := metrics.NewMetric(receivedData.ID, value, receivedData.MType)

	if err != nil {
		h.log.Debugf("Failed to update metric: %v", err)
		w.WriteHeader(status)
		w.Write([]byte(value))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(respData)
	}
}

func NewMetricRegistryHandler(registry storage.Repository, logger log.Logger, minfo MetricURLInfo,
	storageSaver storage.StorageSaver, DBConfig config.DBConfig) *MetricRegistryHandler {
	return &MetricRegistryHandler{registry: registry, log: logger, metricInfo: minfo, storageSaver: storageSaver, databaseConfig: DBConfig}
}

func NewDefaultMetricRegistryHandler(logger log.Logger, registry storage.Repository,
	storageSaver storage.StorageSaver, config config.DBConfig) (*MetricRegistryHandler, error) {
	minfo := MetricURLInfo{
		Name:  "metricName",
		Value: "metricValue",
		Type:  "metricType",
	}

	return NewMetricRegistryHandler(registry, logger, minfo, storageSaver, config), nil
}
