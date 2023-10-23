package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"github.com/go-chi/chi"
)

type MetricURLInfo struct {
	Type  string
	Name  string
	Value string
}

type MetricRegistryHandler struct {
	registry       storage.MetricsStorage
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

func (h *MetricRegistryHandler) getMetric(mtype, mname string) (metricValue string, statusCode int, err error) {
	repo, err := h.registry.GetRepository(mtype)

	if err != nil {
		err = fmt.Errorf("failed to get repository for metric %v: %w", mtype, err)
		return "", http.StatusBadRequest, err
	}

	m, err := repo.Get(mname)

	if err != nil {
		return "", http.StatusNotFound, err
	}

	return m.GetValue(), http.StatusOK, nil
}

func (h *MetricRegistryHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))
	metricName := chi.URLParam(r, h.metricInfo.Name)

	value, status, err := h.getMetric(metricType, metricName)

	if err != nil {
		h.log.Errorf("Failed to get metric of type %v, name %v: %v", metricType, metricName, err)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(value))
}

func (h *MetricRegistryHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := common.MetricJSON{}

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to decode JSON data: %v", err)
		w.Write([]byte("Bad metric format"))
		return
	}

	value, status, err := h.getMetric(receivedData.MType, receivedData.ID)

	if err != nil {
		h.log.Errorf("Failed to get metric of type %v, name %v: %v", receivedData.MType, receivedData.ID, err)
	}

	if status != http.StatusOK {
		w.WriteHeader(status)
		w.Write([]byte(value))
		return
	}

	if err := receivedData.SetData(value); err != nil {
		h.log.Debugf("Failed to get metric %v", err)
		http.Error(w, "Failed to get metric", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(receivedData)
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

func (h *MetricRegistryHandler) updateMetric(mtype, mname, mvalue string) (metricValue string, statusCode int, err error) {
	repo, err := h.registry.GetRepository(mtype)

	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to get repository for metric: %v: %w", mtype, err)
	}

	updatedVal, err := repo.AddOrUpdate(mname, mvalue)

	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to add/update metric %v with value %v: %w", mname, mvalue, err)
	}

	if h.storageSaver != nil {
		err := h.storageSaver.Save()
		if err != nil {
			h.log.Errorf("Failed to update persistent storage: %v", err)
		}
	}

	return updatedVal, http.StatusOK, nil
}

func (h *MetricRegistryHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))
	metricValue := strings.ToLower(chi.URLParam(r, h.metricInfo.Value))
	metricName := chi.URLParam(r, h.metricInfo.Name)

	value, status, err := h.updateMetric(metricType, metricName, metricValue)

	if err != nil {
		h.log.Debugf("Failed to  update metric: %v", err)
	}

	w.WriteHeader(status)
	w.Write([]byte(value))
}

func (h *MetricRegistryHandler) updateMetricsFromJSON(metrics []common.MetricJSON, mtype string) error {
	repo, err := h.registry.GetRepository(mtype)
	if err != nil {
		return fmt.Errorf("failed to get repository for metric: %v: %w", mtype, err)
	}

	err = repo.AddMetricsBulk(metrics)

	if err != nil {
		return fmt.Errorf("failed to get add metrics of type: %v: %w", mtype, err)
	}

	return nil
}

func (h *MetricRegistryHandler) gatherMetrics(metrics []common.MetricJSON, metricsSet map[string]common.MetricJSON) {
	for _, m := range metrics {
		metricsSet[m.ID] = m
	}
}

func (h *MetricRegistryHandler) UpdateMetricsFromJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := make([]common.MetricJSON, 0)

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to decode JSON data: %v", err)
		w.Write([]byte("Bad metric format"))
		return
	}

	gaugeMetrics := make([]common.MetricJSON, 0)
	counterMetrics := make([]common.MetricJSON, 0)

	for _, m := range receivedData {
		currentMetricType := strings.ToLower(m.MType)
		if currentMetricType == common.MetricTypeGauge {
			gaugeMetrics = append(gaugeMetrics, m)
		} else if currentMetricType == common.MetricTypeCounter {
			counterMetrics = append(counterMetrics, m)
		}
	}

	onUpdateError := func(mtype string, err error) {
		h.log.Errorf("failed to get add metrics of type: %v: %v", mtype, err)
		var status int
		var dbErr storage.DatabaseError
		var dataErr storage.BadDataError
		if errors.As(err, &dbErr) {
			status = http.StatusInternalServerError
		} else if errors.As(err, &dataErr) {
			status = http.StatusBadRequest
		}

		http.Error(w, "", status)
	}

	err := h.updateMetricsFromJSON(gaugeMetrics, common.MetricTypeGauge)
	if err != nil {
		onUpdateError(common.MetricTypeGauge, err)
		return
	}

	err = h.updateMetricsFromJSON(counterMetrics, common.MetricTypeCounter)
	if err != nil {
		onUpdateError(common.MetricTypeCounter, err)
		return
	}

	respMetrics := make([]common.MetricJSON, 0, len(gaugeMetrics)+len(counterMetrics))
	respMetrics = append(respMetrics, gaugeMetrics...)
	respMetrics = append(respMetrics, counterMetrics...)

	respMetricsSet := make(map[string]common.MetricJSON, len(gaugeMetrics)+len(counterMetrics))

	h.gatherMetrics(respMetrics, respMetricsSet)
	respMetrics = respMetrics[:len(respMetricsSet)]

	idx := 0
	for _, m := range respMetricsSet {
		respMetrics[idx] = m
		idx++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(respMetrics)
}

func (h *MetricRegistryHandler) UpdateMetricFromJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := common.MetricJSON{}

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to decode JSON data: %v", err)
		w.Write([]byte("Bad metric format"))
		return
	}

	value, err := receivedData.GetData()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Debugf("Failed to get metric data: %v", err)
		w.Write([]byte("Bad metric value"))
		return
	}

	value, status, err := h.updateMetric(receivedData.MType, receivedData.ID, value)

	if err != nil {
		h.log.Debugf("Failed to updatge metric: %v", err)
		w.WriteHeader(status)
		w.Write([]byte(value))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		receivedData.SetData(value)
		json.NewEncoder(w).Encode(receivedData)
	}
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

	updatedVal, err := repo.AddOrUpdate(metricName, metricValue)

	if err != nil {
		h.log.Debugf("Bad metric value: %v. %v", metricValue, err)
		http.Error(w, "Bad metric value", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	logStr := fmt.Sprintf("Metric type '%v', name: '%v', value: '%v' updated. New value: '%v'",
		metricType, metricName, metricValue, updatedVal)
	h.log.Debugf(logStr)
	w.Write([]byte(logStr))
}

func NewMetricRegistryHandler(registry storage.MetricsStorage, logger log.Logger, minfo MetricURLInfo,
	storageSaver storage.StorageSaver, DBConfig config.DBConfig) *MetricRegistryHandler {
	return &MetricRegistryHandler{registry: registry, log: logger, metricInfo: minfo, storageSaver: storageSaver, databaseConfig: DBConfig}
}

func NewDefaultMetricRegistryHandler(logger log.Logger, registry storage.MetricsStorage,
	storageSaver storage.StorageSaver, config config.DBConfig) (*MetricRegistryHandler, error) {
	minfo := MetricURLInfo{
		Name:  "metricName",
		Value: "metricValue",
		Type:  "metricType",
	}

	return NewMetricRegistryHandler(registry, logger, minfo, storageSaver, config), nil
}
