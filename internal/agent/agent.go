package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	monitorHttp "github.com/fuzzy-toozy/metrics-service/internal/agent/http"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type Agent struct {
	metricsMonitor monitor.Monitor
	httpClient     monitorHttp.HTTPClient
	log            log.Logger
	config         config.Config
}

func NewAgent(config config.Config, httpClient monitorHttp.HTTPClient,
	metricsMonitor monitor.Monitor, logger log.Logger) *Agent {
	a := Agent{config: config, httpClient: httpClient, metricsMonitor: metricsMonitor, log: logger}
	return &a
}

func (a *Agent) ReportMetrics() error {
	serverEndpoint := a.config.ServerAddress + a.config.ReportURL
	serverEndpoint = path.Clean(serverEndpoint)
	serverEndpoint = strings.Trim(serverEndpoint, "/")
	serverEndpoint = fmt.Sprintf("http://%v", serverEndpoint)
	var buffer bytes.Buffer
	return a.metricsMonitor.GetMetrics().ForEachMetric(func(metricName string, m metrics.Metric) error {
		metricValue := m.GetValue()
		metricType := m.GetType()
		metricJSON := common.MetricJSON{ID: metricName, MType: metricType}

		if err := metricJSON.SetData(metricValue); err != nil {
			return fmt.Errorf("failed to set metric %v data to %v", metricName, metricValue)
		}

		buffer.Reset()
		if err := json.NewEncoder(&buffer).Encode(metricJSON); err != nil {
			return fmt.Errorf("failed to encode metric to JSON %v", err)
		}

		req, err := http.NewRequest(http.MethodPost, serverEndpoint, &buffer)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := a.httpClient.Send(req)

		if resp != nil {
			defer func() {
				_, err := io.Copy(io.Discard, resp.Body)
				if err != nil {
					a.log.Debugf("Failed reading request body: %v", err)
				}
				resp.Body.Close()
			}()

			a.log.Debugf("Sent metric of type %v, name %v, value %v to %v. Status %v",
				metricType, metricName, metricValue, serverEndpoint, resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("failed to send metric %v. Status code: %v", metricName, resp.StatusCode)
			}
		}

		return err
	})
}

func (a *Agent) Run() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-time.After(2 * time.Second):
			err := a.metricsMonitor.GatherMetrics()
			if err != nil {
				a.log.Warnf("Failed to gather metrics. %v", err)
			}
		case <-ticker.C:
			err := a.ReportMetrics()
			if err != nil {
				a.log.Warnf("Failed to report metric. %v", err)
			}
		}
	}
}
