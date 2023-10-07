package monitor

import (
	"math/rand"
	"runtime"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

var metricsGatherCallbacks []func(*runtime.MemStats, storage.MetricsStorage) error = []func(*runtime.MemStats, storage.MetricsStorage) error{
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("Alloc", metrics.NewGaugeMeric(float64(m.Alloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("TotalAlloc", metrics.NewGaugeMeric(float64(m.TotalAlloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("Sys", metrics.NewGaugeMeric(float64(m.Sys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("Lookups", metrics.NewGaugeMeric(float64(m.Lookups)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("Mallocs", metrics.NewGaugeMeric(float64(m.Mallocs)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("Frees", metrics.NewGaugeMeric(float64(m.Frees)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("HeapAlloc", metrics.NewGaugeMeric(float64(m.HeapAlloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("HeapSys", metrics.NewGaugeMeric(float64(m.HeapSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("HeapIdle", metrics.NewGaugeMeric(float64(m.HeapIdle)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("HeapInuse", metrics.NewGaugeMeric(float64(m.HeapInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("HeapReleased", metrics.NewGaugeMeric(float64(m.HeapReleased)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("HeapObjects", metrics.NewGaugeMeric(float64(m.HeapObjects)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("StackInuse", metrics.NewGaugeMeric(float64(m.StackInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("StackSys", metrics.NewGaugeMeric(float64(m.StackSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("MSpanInuse", metrics.NewGaugeMeric(float64(m.MSpanInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("MSpanSys", metrics.NewGaugeMeric(float64(m.MSpanSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("MCacheInuse", metrics.NewGaugeMeric(float64(m.MCacheInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("MCacheSys", metrics.NewGaugeMeric(float64(m.MCacheSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("BuckHashSys", metrics.NewGaugeMeric(float64(m.BuckHashSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("GCSys", metrics.NewGaugeMeric(float64(m.GCSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("OtherSys", metrics.NewGaugeMeric(float64(m.OtherSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("NextGC", metrics.NewGaugeMeric(float64(m.NextGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("LastGC", metrics.NewGaugeMeric(float64(m.LastGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("PauseTotalNs", metrics.NewGaugeMeric(float64(m.PauseTotalNs)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("NumGC", metrics.NewGaugeMeric(float64(m.NumGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("NumForcedGC", metrics.NewGaugeMeric(float64(m.NumForcedGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("GCCPUFraction", metrics.NewGaugeMeric(float64(m.GCCPUFraction)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("RandomValue", metrics.NewGaugeMeric(float64(rand.Intn(999999))))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate("PollCount", metrics.NewCounterMeric(1))
	},
}

type Monitor interface {
	GatherMetrics() error
	GetMetrics() storage.MetricsStorage
}

type CommonMonitor struct {
	storage storage.MetricsStorage
	log     log.Logger
	metrics runtime.MemStats
}

func (m *CommonMonitor) GatherMetrics() error {
	runtime.ReadMemStats(&m.metrics)
	for _, callback := range metricsGatherCallbacks {
		err := callback(&m.metrics, m.storage)
		if err != nil {
			m.log.Warnf("Failed gathering metrics: %v\n", err)
		}
	}
	return nil
}

func (m *CommonMonitor) GetMetrics() storage.MetricsStorage {
	return m.storage
}

func NewCommonMonitor(s storage.MetricsStorage, l log.Logger) *CommonMonitor {
	return &CommonMonitor{storage: s, log: l}
}
