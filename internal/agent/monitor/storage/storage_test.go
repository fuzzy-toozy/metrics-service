package storage

import (
	"strconv"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/metrics"
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
		require.NoError(t, storage.AddOrUpdate(n, metrics.NewGaugeMeric(v)))
	}

	require.NoError(t, storage.ForEachMetric(func(name string, m metrics.Metric) error {
		v, ok := gaugeMetrics[name]
		require.True(t, ok)
		g, ok := m.(*metrics.GaugeMetric)
		require.True(t, ok)
		require.Equal(t, v, g.Val)

		newVal := v + v
		m.UpdateValue(metrics.NewGaugeMeric(newVal))

		require.Equal(t, newVal, g.Val)
		sVal := strconv.FormatFloat(g.Val, 'f', -1, 64)
		require.Equal(t, sVal, m.GetValue())

		return nil
	}))

	for n := range gaugeMetrics {
		require.NoError(t, storage.Delete(n))
	}

	counterMetrics := map[string]int64{
		"one":   10,
		"two":   11,
		"three": 12,
	}

	for n, v := range counterMetrics {
		require.NoError(t, storage.AddOrUpdate(n, metrics.NewCounterMeric(v)))
	}

	require.NoError(t, storage.ForEachMetric(func(name string, m metrics.Metric) error {
		v, ok := counterMetrics[name]
		require.True(t, ok)
		g, ok := m.(*metrics.CounterMetric)
		require.True(t, ok)
		require.Equal(t, v, g.Val)

		newVal := v + v
		m.UpdateValue(metrics.NewCounterMeric(newVal))

		require.Equal(t, newVal+v, g.Val)
		sVal := strconv.FormatInt(g.Val, 10)
		require.Equal(t, sVal, m.GetValue())

		return nil
	}))
}