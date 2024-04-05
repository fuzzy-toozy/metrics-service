package agent

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
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

func BenchmarkReportMetrics(b *testing.B) {
	logger := log.NewDevZapLogger()
	os.Args = os.Args[0:1]
	c, err := config.BuildConfig()
	if err != nil {
		logger.Errorf("Failed to build config: %v", err)
		return
	}

	dummyClient := DummyClient{}

	w := newWorker(c, logger, dummyClient)

	m := monitor.NewMetricsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())

	m.GatherMetrics()

	allMetrics := m.GetMetricsStorage().GetAllMetrics()

	rData := reportData{}
	rData.data = allMetrics
	rData.dType = tBULK

	for i := 0; i < b.N; i++ {
		w.reportDataJSON(rData)
	}
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

	w := newWorker(c, logger, dummyClient)

	rData := reportData{}
	rData.data = storage.StorageMetric(metrics.NewCounterMetric("metric", 1000))
	rData.dType = tSINGLE

	for i := 0; i < b.N; i++ {
		w.reportDataJSON(rData)
	}
}
