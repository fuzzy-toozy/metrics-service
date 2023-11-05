package storage

import (
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/stretchr/testify/require"
)

func TestMetricsStorage(t *testing.T) {
	storage := NewCommonMetricsStorage()
	gaugeMetrics := map[string]float64{
		"one":   10.10,
		"two":   11.11,
		"three": 12.12,
	}

	for n, v := range gaugeMetrics {
		require.NoError(t, storage.AddOrUpdate(metrics.NewGaugeMetric(n, v)))
	}

	for _, m := range storage.GetAllMetrics() {
		v, ok := gaugeMetrics[m.ID]
		require.True(t, ok)
		require.Equal(t, metrics.GaugeMetricType, m.MType)
		require.NotNil(t, m.Value)
		require.Equal(t, v, *m.Value)
	}

	storage = NewCommonMetricsStorage()
	counterMetrics := map[string]int64{
		"one":   10,
		"two":   11,
		"three": 12,
	}

	for n, v := range counterMetrics {
		require.NoError(t, storage.AddOrUpdate(metrics.NewCounterMetric(n, v)))
	}

	for _, m := range storage.GetAllMetrics() {
		v, ok := counterMetrics[m.ID]
		require.True(t, ok)
		require.Equal(t, metrics.CounterMetricType, m.MType)
		require.NotNil(t, m.Delta)
		require.Equal(t, v, *m.Delta)
	}
}
