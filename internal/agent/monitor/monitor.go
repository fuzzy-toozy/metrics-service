package monitor

import (
	"math/rand"
	"runtime"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
)

var metricsGatherCallbacks []func(*runtime.MemStats, storage.MetricsStorage) error = []func(*runtime.MemStats, storage.MetricsStorage) error{
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("Alloc", float64(m.Alloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("TotalAlloc", float64(m.TotalAlloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("Sys", float64(m.Sys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("Lookups", float64(m.Lookups)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("Mallocs", float64(m.Mallocs)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("Frees", float64(m.Frees)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("HeapAlloc", float64(m.HeapAlloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("HeapSys", float64(m.HeapSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("HeapIdle", float64(m.HeapIdle)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("HeapInuse", float64(m.HeapInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("HeapReleased", float64(m.HeapReleased)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("HeapObjects", float64(m.HeapObjects)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("StackInuse", float64(m.StackInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("StackSys", float64(m.StackSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("MSpanInuse", float64(m.MSpanInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("MSpanSys", float64(m.MSpanSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("MCacheInuse", float64(m.MCacheInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("MCacheSys", float64(m.MCacheSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("BuckHashSys", float64(m.BuckHashSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("GCSys", float64(m.GCSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("OtherSys", float64(m.OtherSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("NextGC", float64(m.NextGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("LastGC", float64(m.LastGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("PauseTotalNs", float64(m.PauseTotalNs)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("NumGC", float64(m.NumGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("NumForcedGC", float64(m.NumForcedGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("GCCPUFraction", float64(m.GCCPUFraction)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric("RandomValue", float64(rand.Intn(999999))))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewCounterMetric("PollCount", 1))
	},
}

type Monitor interface {
	GatherMetrics() error
	GetMetricsStorage() storage.MetricsStorage
}

type CommonMonitor struct {
	storage storage.MetricsStorage
	log     log.Logger
	metrics runtime.MemStats
}

func (m *CommonMonitor) GatherMetrics() error {
	m.storage.Clear()
	runtime.ReadMemStats(&m.metrics)
	for _, callback := range metricsGatherCallbacks {
		err := callback(&m.metrics, m.storage)
		if err != nil {
			m.log.Warnf("Failed gathering metrics: %v\n", err)
		}
	}
	return nil
}

func (m *CommonMonitor) GetMetricsStorage() storage.MetricsStorage {
	return m.storage
}

func (m *CommonMonitor) GetMetrics() storage.MetricsStorage {
	return m.storage
}

func NewCommonMonitor(s storage.MetricsStorage, l log.Logger) *CommonMonitor {
	return &CommonMonitor{storage: s, log: l}
}
