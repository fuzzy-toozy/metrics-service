package monitor

import (
	"math/rand"
	"runtime"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

var metricsGatherCallbacks []func(*runtime.MemStats, MetricsStorage) error = []func(*runtime.MemStats, MetricsStorage) error{
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("Alloc", NewGaugeMeric(float64(m.Alloc)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("TotalAlloc", NewGaugeMeric(float64(m.TotalAlloc)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("Sys", NewGaugeMeric(float64(m.Sys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("Lookups", NewGaugeMeric(float64(m.Lookups)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("Mallocs", NewGaugeMeric(float64(m.Mallocs)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("Frees", NewGaugeMeric(float64(m.Frees)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("HeapAlloc", NewGaugeMeric(float64(m.HeapAlloc)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("HeapSys", NewGaugeMeric(float64(m.HeapSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("HeapIdle", NewGaugeMeric(float64(m.HeapIdle)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("HeapInuse", NewGaugeMeric(float64(m.HeapInuse)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("HeapReleased", NewGaugeMeric(float64(m.HeapReleased)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("HeapObjects", NewGaugeMeric(float64(m.HeapObjects)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("StackInuse", NewGaugeMeric(float64(m.StackInuse)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("StackSys", NewGaugeMeric(float64(m.StackSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("MSpanInuse", NewGaugeMeric(float64(m.MSpanInuse)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("MSpanSys", NewGaugeMeric(float64(m.MSpanSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("MCacheInuse", NewGaugeMeric(float64(m.MCacheInuse)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("MCacheSys", NewGaugeMeric(float64(m.MCacheSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("BuckHashSys", NewGaugeMeric(float64(m.BuckHashSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("GCSys", NewGaugeMeric(float64(m.GCSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("OtherSys", NewGaugeMeric(float64(m.OtherSys)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("NextGC", NewGaugeMeric(float64(m.NextGC)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("LastGC", NewGaugeMeric(float64(m.LastGC)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("PauseTotalNs", NewGaugeMeric(float64(m.PauseTotalNs)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("NumGC", NewGaugeMeric(float64(m.NumGC)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("NumForcedGC", NewGaugeMeric(float64(m.NumForcedGC)))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("RandomValue", NewGaugeMeric(float64(rand.Intn(999999))))
	},
	func(m *runtime.MemStats, s MetricsStorage) error {
		return s.AddOrUpdate("PollCount", NewCounterMeric(1))
	},
}

type Monitor interface {
	GatherMetrics() error
	GetMetrics() MetricsStorage
}

type CommonMonitor struct {
	storage MetricsStorage
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

func (m *CommonMonitor) GetMetrics() MetricsStorage {
	return m.storage
}

func NewCommonMonitor(s MetricsStorage, l log.Logger) *CommonMonitor {
	return &CommonMonitor{storage: s, log: l}
}
