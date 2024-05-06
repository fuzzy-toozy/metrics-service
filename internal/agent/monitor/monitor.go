// Package monitor Defines monitors that fetch agent's metrics.
package monitor

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

const (
	AllocMetricName         = "Alloc"
	TotalAllocMetricName    = "TotalAlloc"
	SysMetricName           = "Sys"
	LookupsMetricName       = "Lookups"
	MallocsMetricName       = "Mallocs"
	FreesMetricName         = "Frees"
	HeapAllocMetricName     = "HeapAlloc"
	HeapSysMetricName       = "HeapSys"
	HeapIdleMetricName      = "HeapIdle"
	HeapInuseMetricName     = "HeapInuse"
	HeapReleasedMetricName  = "HeapReleased"
	HeapObjectsMetricName   = "HeapObjects"
	StackInuseMetricName    = "StackInuse"
	StackSysMetricName      = "StackSys"
	MSpanInuseMetricName    = "MSpanInuse"
	MSpanSysMetricName      = "MSpanSys"
	MCacheInuseMetricName   = "MCacheInuse"
	MCacheSysMetricName     = "MCacheSys"
	BuckHashSysMetricName   = "BuckHashSys"
	GCSysMetricName         = "GCSys"
	OtherSysMetricName      = "OtherSys"
	NextGCMetricName        = "NextGC"
	LastGCMetricName        = "LastGC"
	PauseTotalNsMetricName  = "PauseTotalNs"
	NumGCMetricName         = "NumGC"
	NumForcedGCMetricName   = "NumForcedGC"
	GCCPUFractionMetricName = "GCCPUFraction"
	RandomValueMetricName   = "RandomValue"
	PollCountMetricName     = "PollCount"
)

const (
	PsTotalMemMetric = "TotalMemory"
	PsFreeMemMetric  = "FreeMemory"
	PsCPUUtilMetric  = "CPUutilization"
)

var metricTypeMap = map[string]string{
	AllocMetricName:         metrics.GaugeMetricType,
	TotalAllocMetricName:    metrics.GaugeMetricType,
	SysMetricName:           metrics.GaugeMetricType,
	LookupsMetricName:       metrics.GaugeMetricType,
	MallocsMetricName:       metrics.GaugeMetricType,
	FreesMetricName:         metrics.GaugeMetricType,
	HeapAllocMetricName:     metrics.GaugeMetricType,
	HeapSysMetricName:       metrics.GaugeMetricType,
	HeapIdleMetricName:      metrics.GaugeMetricType,
	HeapInuseMetricName:     metrics.GaugeMetricType,
	HeapReleasedMetricName:  metrics.GaugeMetricType,
	HeapObjectsMetricName:   metrics.GaugeMetricType,
	StackInuseMetricName:    metrics.GaugeMetricType,
	StackSysMetricName:      metrics.GaugeMetricType,
	MSpanInuseMetricName:    metrics.GaugeMetricType,
	MSpanSysMetricName:      metrics.GaugeMetricType,
	MCacheInuseMetricName:   metrics.GaugeMetricType,
	MCacheSysMetricName:     metrics.GaugeMetricType,
	BuckHashSysMetricName:   metrics.GaugeMetricType,
	GCSysMetricName:         metrics.GaugeMetricType,
	OtherSysMetricName:      metrics.GaugeMetricType,
	NextGCMetricName:        metrics.GaugeMetricType,
	LastGCMetricName:        metrics.GaugeMetricType,
	PauseTotalNsMetricName:  metrics.GaugeMetricType,
	NumGCMetricName:         metrics.GaugeMetricType,
	NumForcedGCMetricName:   metrics.GaugeMetricType,
	GCCPUFractionMetricName: metrics.GaugeMetricType,
	RandomValueMetricName:   metrics.GaugeMetricType,
	PollCountMetricName:     metrics.CounterMetricType,

	PsTotalMemMetric: metrics.GaugeMetricType,
	PsFreeMemMetric:  metrics.GaugeMetricType,
}

var metricsGatherCallbacks []func(*runtime.MemStats, storage.MetricsStorage) error = []func(*runtime.MemStats, storage.MetricsStorage) error{
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(AllocMetricName, float64(m.Alloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(TotalAllocMetricName, float64(m.TotalAlloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(SysMetricName, float64(m.Sys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(LookupsMetricName, float64(m.Lookups)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(MallocsMetricName, float64(m.Mallocs)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(FreesMetricName, float64(m.Frees)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(HeapAllocMetricName, float64(m.HeapAlloc)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(HeapSysMetricName, float64(m.HeapSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(HeapIdleMetricName, float64(m.HeapIdle)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(HeapInuseMetricName, float64(m.HeapInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(HeapReleasedMetricName, float64(m.HeapReleased)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(HeapObjectsMetricName, float64(m.HeapObjects)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(StackInuseMetricName, float64(m.StackInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(StackSysMetricName, float64(m.StackSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(MSpanInuseMetricName, float64(m.MSpanInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(MSpanSysMetricName, float64(m.MSpanSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(MCacheInuseMetricName, float64(m.MCacheInuse)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(MCacheSysMetricName, float64(m.MCacheSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(BuckHashSysMetricName, float64(m.BuckHashSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(GCSysMetricName, float64(m.GCSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(OtherSysMetricName, float64(m.OtherSys)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(NextGCMetricName, float64(m.NextGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(LastGCMetricName, float64(m.LastGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(PauseTotalNsMetricName, float64(m.PauseTotalNs)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(NumGCMetricName, float64(m.NumGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(NumForcedGCMetricName, float64(m.NumForcedGC)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		return s.AddOrUpdate(metrics.NewGaugeMetric(GCCPUFractionMetricName, float64(m.GCCPUFraction)))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		const randCap = 999999
		return s.AddOrUpdate(metrics.NewGaugeMetric(RandomValueMetricName, float64(rand.Intn(randCap))))
	},
	func(m *runtime.MemStats, s storage.MetricsStorage) error {
		const StartVal = 1
		return s.AddOrUpdate(metrics.NewCounterMetric(PollCountMetricName, StartVal))
	},
}

// Monitor describes interface of metrics fetcher.
type Monitor interface {
	GatherMetrics() error
	GetMetricsStorage() storage.MetricsStorage
}

// CommonMonitor desfault monitor implementation.
// Gathers golang app memory metrics.
type CommonMonitor struct {
	storage storage.MetricsStorage
	log     log.Logger
	metrics runtime.MemStats
}

// GatherMetrics fetches go memory metrics from local system and add them to storage.
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

// GetMetricsStorage return underlying metrics storage.
func (m *CommonMonitor) GetMetricsStorage() storage.MetricsStorage {
	return m.storage
}

func NewMetricsMonitor(s storage.MetricsStorage, l log.Logger) *CommonMonitor {
	return &CommonMonitor{storage: s, log: l}
}

// PsMonitor system memory and CPU metrics frol local system.
type PsMonitor struct {
	storage storage.MetricsStorage
	log     log.Logger
}

func NewPsMonitor(s storage.MetricsStorage, l log.Logger) *PsMonitor {
	return &PsMonitor{storage: s, log: l}
}

// GetMetricsStorage return underlying metrics storage.
func (m *PsMonitor) GetMetricsStorage() storage.MetricsStorage {
	return m.storage
}

// GatherMetrics fetches system memory and CPU metrics from local system and add them to storage.
func (m *PsMonitor) GatherMetrics() error {
	m.storage.Clear()
	memData, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	err = m.storage.AddOrUpdate(metrics.NewGaugeMetric(PsTotalMemMetric, float64(memData.Total)))
	if err != nil {
		return err
	}

	err = m.storage.AddOrUpdate(metrics.NewGaugeMetric(PsFreeMemMetric, float64(memData.Available)))
	if err != nil {
		return err
	}

	const usageInterval = 100
	cpuLoad, err := cpu.Percent(usageInterval*time.Millisecond, true)

	if err != nil {
		return err
	}

	for i, val := range cpuLoad {
		err = m.storage.AddOrUpdate(metrics.NewGaugeMetric(fmt.Sprintf("%v%v", PsCPUUtilMetric, i+1), val))
		if err != nil {
			return err
		}
	}

	return nil
}
