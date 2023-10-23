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
	"github.com/fuzzy-toozy/metrics-service/internal/compression"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type Agent struct {
	metricsMonitor    monitor.Monitor
	httpClient        monitorHttp.HTTPClient
	log               log.Logger
	config            config.Config
	buffer            bytes.Buffer
	compressionBuffer bytes.Buffer
	compressAlgo      string
}

func NewAgent(config config.Config, httpClient monitorHttp.HTTPClient,
	metricsMonitor monitor.Monitor, logger log.Logger) *Agent {
	a := Agent{config: config,
		httpClient:        httpClient,
		metricsMonitor:    metricsMonitor,
		log:               logger,
		buffer:            bytes.Buffer{},
		compressionBuffer: bytes.Buffer{},
		compressAlgo:      "gzip"}
	return &a
}

func (a *Agent) GetCompressedBytes(data []byte) ([]byte, error) {
	if len(a.compressAlgo) > 0 {
		factory, err := compression.GetCompressorFactory(a.compressAlgo)
		if err != nil {
			return nil, fmt.Errorf("failed to get compression factory: %w", err)
		}
		a.compressionBuffer.Reset()
		compressor, err := factory(&a.compressionBuffer)
		if err != nil {
			return nil, fmt.Errorf("failed to create compressor: %w", err)
		}

		_, err = compressor.Write(a.buffer.Bytes())

		if err != nil {
			return nil, fmt.Errorf("failed to compress data: %w", err)
		}

		err = compressor.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to finalize compressor: %w", err)
		}

		return a.compressionBuffer.Bytes(), nil
	}

	return nil, fmt.Errorf("no compression algo specified")
}

func (a *Agent) ReportMetricsBulk() error {
	serverEndpoint := a.config.ServerAddress + a.config.ReportBulkURL
	serverEndpoint = path.Clean(serverEndpoint)
	serverEndpoint = strings.Trim(serverEndpoint, "/")
	serverEndpoint = fmt.Sprintf("http://%v", serverEndpoint)

	metricsList := make([]common.MetricJSON, 0)

	err := a.metricsMonitor.GetMetrics().ForEachMetric(func(metricName string, m metrics.Metric) error {
		metricValue := m.GetValue()
		metricType := m.GetType()
		metricJSON := common.MetricJSON{ID: metricName, MType: metricType}

		if err := metricJSON.SetData(metricValue); err != nil {
			return fmt.Errorf("failed to set metric %v data to %v: %w", metricName, metricValue, err)
		}

		metricsList = append(metricsList, metricJSON)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to gather metrics: %w", err)
	}

	contentType := "application/json"
	contentEncoding := ""

	a.buffer.Reset()
	if err := json.NewEncoder(&a.buffer).Encode(metricsList); err != nil {
		return fmt.Errorf("failed to encode metrics to JSON: %w", err)
	}

	bytesToSend := a.buffer.Bytes()

	compressedBytes, err := a.GetCompressedBytes(bytesToSend)
	if err != nil {
		a.log.Debugf("Unable to enable compression: %v", err)
	} else {
		bytesToSend = compressedBytes
		contentEncoding = a.compressAlgo
	}

	req, err := http.NewRequest(http.MethodPost, serverEndpoint, bytes.NewBuffer(bytesToSend))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Encoding", contentEncoding)

	resp, err := a.httpClient.Send(req)

	if resp != nil {
		defer func() {
			_, err := io.Copy(io.Discard, resp.Body)
			if err != nil {
				a.log.Debugf("Failed reading request body: %v", err)
			}
			resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("failed to send metrics. Status code: %v", resp.StatusCode)
		}
	}

	return err
}

func (a *Agent) ReportMetrics() error {
	serverEndpoint := a.config.ServerAddress + a.config.ReportURL
	serverEndpoint = path.Clean(serverEndpoint)
	serverEndpoint = strings.Trim(serverEndpoint, "/")
	serverEndpoint = fmt.Sprintf("http://%v", serverEndpoint)

	return a.metricsMonitor.GetMetrics().ForEachMetric(func(metricName string, m metrics.Metric) error {
		metricValue := m.GetValue()
		metricType := m.GetType()
		metricJSON := common.MetricJSON{ID: metricName, MType: metricType}
		contentType := "application/json"
		contentEncoding := ""

		if err := metricJSON.SetData(metricValue); err != nil {
			return fmt.Errorf("failed to set metric %v data to %v: %w", metricName, metricValue, err)
		}

		a.buffer.Reset()
		if err := json.NewEncoder(&a.buffer).Encode(metricJSON); err != nil {
			return fmt.Errorf("failed to encode metric to JSON: %w", err)
		}

		bytesToSend := a.buffer.Bytes()

		compressedBytes, err := a.GetCompressedBytes(bytesToSend)
		if err != nil {
			a.log.Debugf("Unable to enable compression: %v", err)
		} else {
			bytesToSend = compressedBytes
			contentEncoding = a.compressAlgo
		}

		req, err := http.NewRequest(http.MethodPost, serverEndpoint, bytes.NewBuffer(bytesToSend))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Content-Encoding", contentEncoding)

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
	retryExecutor := common.NewCommonRetryExecutor(2*time.Second, 3, nil)

	for {
		select {
		case <-time.After(2 * time.Second):
			err := a.metricsMonitor.GatherMetrics()
			if err != nil {
				a.log.Warnf("Failed to gather metrics. %v", err)
			}
		case <-ticker.C:

			err := retryExecutor.RetryOnError(func() error {
				return a.ReportMetricsBulk()
			})
			if err != nil {
				a.log.Warnf("Failed to report metrics bulk. %v", err)
			}
			err = retryExecutor.RetryOnError(func() error {
				return a.ReportMetrics()
			})
			if err != nil {
				a.log.Warnf("Failed to report metrics. %v", err)
			}
		}
	}
}
