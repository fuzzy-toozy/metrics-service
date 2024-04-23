package monitor

import (
	"runtime"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

func BenchmarkCommonMonitorGatherMetrics(b *testing.B) {
	m := NewMetricsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())

	var err error
	for i := 0; i < b.N; i++ {
		err = m.GatherMetrics()
	}
	runtime.KeepAlive(err)
}

func BenchmarkPsMonitorGatherMetrics(b *testing.B) {
	m := NewPsMonitor(storage.NewCommonMetricsStorage(), log.NewDevZapLogger())

	var err error
	for i := 0; i < b.N; i++ {
		err = m.GatherMetrics()
	}
	runtime.KeepAlive(err)
}
