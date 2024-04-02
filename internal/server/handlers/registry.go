// Provides handlers to access metrics storage.
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

func setJSONContent(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func respEmptyJSON(w http.ResponseWriter, status int) {
	setJSONContent(w)
	w.WriteHeader(status)
	w.Write([]byte("{}"))
}

func respMetricJSON(m metrics.Metric, w http.ResponseWriter, status int) {
	setJSONContent(w)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(m)
}

func respMetricsJSON(m []metrics.Metric, w http.ResponseWriter, status int) {
	setJSONContent(w)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(m)
}

// @Summary Health Check
// @Description Pings the database to check its availability.
// @Tags Health
// @Produce plain
// @Success 200 {string} string "OK"
// @Failure 500 {string} string
// @Router /ping [get]
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

// @Summary Get Metric
// @Description searches metric by id and type and returns it's value in plain text.
// @Tags Metrics
// @ID get-metric
// @Accept plain
// @Produce plain
// @Param metricName path string true "Name of the metric to retrieve"
// @Param metricType path string true "Type of the metric to retrieve"
// @Success 200 {string} string "Mertic value"
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @Router /value/{metricType}/{metricName} [get]
func (h *MetricRegistryHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := strings.ToLower(chi.URLParam(r, h.metricInfo.Type))

	metricName := chi.URLParam(r, h.metricInfo.Name)

	val, status, err := h.getMetric(metricName, metricType)

	if err != nil {
		h.log.Debugf("failed to get metric: %v", err)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(val))
}

// @Summary Get Metric JSON
// @Description Gets requested metric by id and type and returns it's id, type and value in JSON format.
// @Tags Metrics
// @ID get-metric-json
// @Accept json
// @Produce json
// @Param metric body metrics.Metric true "Metric data"
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /value [post]
//
// Request data example:
// Counter type:
//
//	{
//		 "id":"13eee119-cfaf-4b61-b101-41e26670a069",
//		 "type":"counter",
//	}
//
// Gauge type:
//
//	{
//		 "id":"13eee119-cfaf-4b61-b101-41e26670a069",
//		 "type":"gauge",
//	}
//
// Returned data example:
// Counter metric:
//
//	{
//		 "id":"13eee119-cfaf-4b61-b101-41e26670a069",
//		 "type":"counter",
//		 "delta":42
//	}
//
// Gauge metric:
// Counter metric:
//
//	{
//		 "id":"13eee119-cfaf-4b61-b101-41e26670a069",
//		 "type":"gauge",
//		 "value":0.42
//	}
func (h *MetricRegistryHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := metrics.Metric{}

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		h.log.Debugf("Failed to decode JSON data: %v", err)
		respEmptyJSON(w, http.StatusBadRequest)
		return
	}

	value, status, err := h.getMetric(receivedData.ID, receivedData.MType)

	if err != nil {
		h.log.Debugf("Failed to get metric of type %v, name %v: %v", receivedData.MType, receivedData.ID, err)
	}

	if status != http.StatusOK {
		respEmptyJSON(w, status)
		return
	}

	respData, _ := metrics.NewMetric(receivedData.ID, value, receivedData.MType)

	respMetricJSON(respData, w, status)
}

// @Summary Get All Metrics
// @Description Returns all stored metrics in an HTML table.
// @Tags Metrics
// @ID get-all-metrics
// @Accept html
// @Produce html
// @Success 200 {string} string "<!DOCTYPE html>..."
// @Failure 500 {string} string ""
// @Router / [get]
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

	w.Header().Set("Content-Type", "text/html")
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

// @Summary Update Metric
// @Description Updates the specified metric with the provided value.
// @Tags Metrics
// @ID update-metric
// @Accept plain
// @Produce plain
// @Param metricName path string true "Name of the metric to update"
// @Param metricType path string true "Type of the metric to update"
// @Param metricValue path string true "Value to update the metric with"
// @Success 200 {string} string "Updated metric value"
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @Router /update/{metricType}/{metricName}/{metricValue} [post]
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

// UpdateMetricsFromJSON updates or adds metrics received in request.
//
// @Summary Update or add metrics from JSON
// @Description Updates or adds metrics received in request and returns the updated metrics.
// @Tags Metrics
// @ID update-metrics-from-json
// @Accept json
// @Produce json
// @Param data body []metrics.Metric true "Metrics data"
// @Success 200 {array} metrics.Metric "Updated metrics"
// @Failure 400
// @Failure 500
// @Router /updates [post]
//
// Request data example:
//
//		[{
//			 "id":"13eee119-cfaf-4b61-b101-41e26670a069",
//			 "type":"counter",
//	         "delta": 21
//		},
//		{
//			 "id":"13eee119-cfaf-4b61-b101-41e26670a021",
//			 "type":"gauge",
//		     "value": 21.11
//		}]
//
// Returned data example:
//
//		[{
//			 "id":"13eee119-cfaf-4b61-b101-41e26670a069",
//			 "type":"counter",
//	         "delta": 43
//		},
//		{
//			 "id":"13eee119-cfaf-4b61-b101-41e26670a021",
//			 "type":"gauge",
//		     "value": 21.11
//		}]
func (h *MetricRegistryHandler) UpdateMetricsFromJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := make([]metrics.Metric, 0)
	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		h.log.Debugf("Failed to decode JSON data: %v", err)
		respEmptyJSON(w, http.StatusBadRequest)
		return
	}

	err := h.registry.AddMetricsBulk(receivedData)
	status := errtypes.ErrorToStatus(err)
	if err != nil {
		h.log.Errorf("Failed to add metrics: %v", err)
		respEmptyJSON(w, status)
		return
	}

	respMetricsJSON(receivedData, w, status)
}

// UpdateMetricFromJSON updates or adds metrics received in request.
//
// @Summary Update or add metrics from JSON
// @Description Updates or adds metrics received in request and returns the updated metrics.
// @Tags Metrics
// @ID update-metric-from-json
// @Accept json
// @Produce json
// @Param data body metrics.Metric true "Metrics data"
// @Success 200 {object} metrics.Metric "Updated metric"
// @Failure 400
// @Failure 500
// @Router /update [post]
//
// Request data example:
// Counter type:
//
//	{
//		"id":"13eee119-cfaf-4b61-b101-41e26670a069",
//		"type":"counter",
//		"delta": 21
//	}
//
// Gauge type:
//
//	{
//		"id":"13eee119-cfaf-4b61-b101-41e26670a021",
//		"type":"gauge",
//		"value": 42.12
//	}
//
// Returned data example:
// Counter type:
//
//	{
//		"id":"13eee119-cfaf-4b61-b101-41e26670a069",
//		"type":"counter",
//		"delta": 42
//	}
//
// Gauge type:
//
//	{
//		"id":"13eee119-cfaf-4b61-b101-41e26670a021",
//		"type":"gauge",
//		"value": 42.12
//	}
func (h *MetricRegistryHandler) UpdateMetricFromJSON(w http.ResponseWriter, r *http.Request) {
	receivedData := metrics.Metric{}

	if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
		h.log.Debugf("Failed to decode JSON data: %v", err)
		respEmptyJSON(w, http.StatusBadRequest)
		return
	}

	value, err := receivedData.GetData()
	if err != nil {
		h.log.Debugf("Failed to get metric data: %v", err)
		respEmptyJSON(w, http.StatusBadRequest)
		return
	}

	value, status, err := h.updateMetric(receivedData.MType, receivedData.ID, value)

	if err != nil {
		h.log.Debugf("Failed to update metric: %v", err)
		respEmptyJSON(w, status)
		return
	}

	respData, _ := metrics.NewMetric(receivedData.ID, value, receivedData.MType)
	respMetricJSON(respData, w, status)
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
