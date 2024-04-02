package monitor

import (
	"strings"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/stretchr/testify/require"
)

func monitorTest(mon Monitor, t *testing.T) {
	require.NoError(t, mon.GatherMetrics())

	for _, met := range mon.GetMetricsStorage().GetAllMetrics() {
		if strings.HasPrefix(met.ID, PsCPUUtilMetric) {
			require.Equal(t, met.MType, metrics.GaugeMetricType)
			continue
		}
		mtype, ok := metricTypeMap[met.ID]
		require.True(t, ok)
		require.Equal(t, mtype, met.MType)
	}
}

func Test_MetricsMonitor(t *testing.T) {
	m := NewMetricsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())
	monitorTest(m, t)
}

func Test_PsMonitor(t *testing.T) {
	m := NewPsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())
	monitorTest(m, t)
}
