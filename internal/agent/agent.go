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
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/compression"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
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

	metricsList := a.metricsMonitor.GetMetricsStorage().GetAllMetrics()

	contentType := "application/json"
	contentEncoding := ""

	a.buffer.Reset()
	if err := json.NewEncoder(&a.buffer).Encode(metricsList); err != nil {
		return fmt.Errorf("failed to encode metrics to JSON: %w", err)
	}

	bytesToSend := a.buffer.Bytes()

	var sigHash string
	if a.config.SecretKey != nil {
		hash, err := encryption.SignData(bytesToSend, a.config.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to sign request data: %w", err)
		}
		sigHash = hash
	}

	compressedBytes, err := a.GetCompressedBytes(bytesToSend)
	if err != nil {
		a.log.Debugf("Unable to enable compression: %v", err)
	} else {
		bytesToSend = compressedBytes
		contentEncoding = a.compressAlgo
	}

	req, err := http.NewRequest(http.MethodPost, serverEndpoint, bytes.NewBuffer(bytesToSend))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Encoding", contentEncoding)

	if len(sigHash) != 0 {
		req.Header.Set("HashSHA256", sigHash)
	}

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

	for _, m := range a.metricsMonitor.GetMetricsStorage().GetAllMetrics() {
		contentType := "application/json"
		contentEncoding := ""

		a.buffer.Reset()
		if err := json.NewEncoder(&a.buffer).Encode(m); err != nil {
			return fmt.Errorf("failed to encode metric to JSON: %w", err)
		}

		bytesToSend := a.buffer.Bytes()

		var sigHash string
		if a.config.SecretKey != nil {
			hash, err := encryption.SignData(bytesToSend, a.config.SecretKey)
			if err != nil {
				return fmt.Errorf("failed to sign request data: %w", err)
			}
			sigHash = hash
		}

		compressedBytes, err := a.GetCompressedBytes(bytesToSend)
		if err != nil {
			a.log.Debugf("Unable to enable compression: %v", err)
		} else {
			bytesToSend = compressedBytes
			contentEncoding = a.compressAlgo
		}

		req, err := http.NewRequest(http.MethodPost, serverEndpoint, bytes.NewBuffer(bytesToSend))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Content-Encoding", contentEncoding)

		if len(sigHash) > 0 {
			req.Header.Set("HashSHA256", sigHash)
		}

		resp, err := a.httpClient.Send(req)

		if resp != nil {
			defer func() {
				_, err := io.Copy(io.Discard, resp.Body)
				if err != nil {
					a.log.Debugf("Failed reading request body: %v", err)
				}
				resp.Body.Close()
			}()

			val, _ := m.GetData()
			a.log.Debugf("Sent metric of type %v, name %v, value %v to %v. Status %v",
				m.MType, m.ID, val, serverEndpoint, resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("failed to send metric %v. Status code: %v", m.ID, resp.StatusCode)
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
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
