package monitor

import (
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

func BenchmarkCommonMonitorGatherMetrics(b *testing.B) {
	m := NewMetricsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())

	for i := 0; i < b.N; i++ {
		m.GatherMetrics()
	}
}

func BenchmarkPsMonitorGatherMetrics(b *testing.B) {
	m := NewPsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())

	for i := 0; i < b.N; i++ {
		m.GatherMetrics()
	}
}
