package agent

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"os"
	"runtime"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/worker"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
)

type DummyClient struct {
}

func (c DummyClient) Send(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bufio.NewReader(bytes.NewBuffer(nil))),
	}, nil
}

func (c DummyClient) SetTransport(t http.RoundTripper) {
}

func BenchmarkReportMetrics(b *testing.B) {
	logger := log.NewDevZapLogger()
	os.Args = os.Args[0:1]
	c, err := config.BuildConfig()
	if err != nil {
		logger.Errorf("Failed to build config: %v", err)
		return
	}

	dummyClient := DummyClient{}

	w := worker.NewWorkerHTTP(c, logger, dummyClient)

	m := monitor.NewMetricsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())

	err = m.GatherMetrics()
	if err != nil {
		logger.Errorf("Failed to gather metrics: %v", err)
		return
	}

	allMetrics := m.GetMetricsStorage().GetAllMetrics()

	rData := worker.ReportData{
		Data:  allMetrics,
		DType: worker.BULK,
	}

	for i := 0; i < b.N; i++ {
		err = w.ReportData(rData)
	}
	runtime.KeepAlive(err)
}

func BenchmarkReportMetric(b *testing.B) {
	logger := log.NewDevZapLogger()
	os.Args = os.Args[0:1]
	c, err := config.BuildConfig()
	if err != nil {
		logger.Errorf("Failed to build config: %v", err)
		return
	}

	dummyClient := DummyClient{}

	w := worker.NewWorkerHTTP(c, logger, dummyClient)

	rData := worker.ReportData{
		Data:  storage.StorageMetric(metrics.NewCounterMetric("metric", 1000)),
		DType: worker.SINGLE,
	}

	for i := 0; i < b.N; i++ {
		err = w.ReportData(rData)
	}
	runtime.KeepAlive(err)
}
