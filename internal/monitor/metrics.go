package monitor

import "github.com/fuzzy-toozy/metrics-service/internal/common"

type MetricValType int

type Metric interface {
	GetValue() string
	GetType() string
	UpdateValue(v Metric)
}

type GaugeMetric struct {
	common.Float
}

func (m GaugeMetric) GetType() string {
	return "gauge"
}

func (m *GaugeMetric) UpdateValue(v Metric) {
	metric := v.(*GaugeMetric)
	m.Val = metric.Val
}

type CounterMetric struct {
	common.Int
}

func (m CounterMetric) GetType() string {
	return "counter"
}

func (m *CounterMetric) UpdateValue(v Metric) {
	metric := v.(*CounterMetric)
	m.Val += metric.Val
}

type MetricsStorage interface {
	AddOrUpdate(name string, m Metric) error
	Delete(name string) error
	ForEachMetric(callback func(name string, m Metric) error) error
}

type CommonMetricsStorage struct {
	storage map[string]Metric
}

func (s *CommonMetricsStorage) AddOrUpdate(name string, m Metric) error {
	s.storage[name] = m
	return nil
}

func (s *CommonMetricsStorage) Delete(name string) error {
	delete(s.storage, name)
	return nil
}

func (s *CommonMetricsStorage) ForEachMetric(callback func(name string, m Metric) error) error {
	for n, m := range s.storage {
		err := callback(n, m)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewCommonMetricsStorage() *CommonMetricsStorage {
	s := CommonMetricsStorage{storage: make(map[string]Metric)}
	return &s
}

func NewGaugeMeric(v float64) *GaugeMetric {
	m := GaugeMetric{Float: common.Float{Val: v}}
	return &m
}

func NewCounterMeric(v int64) *CounterMetric {
	m := CounterMetric{Int: common.Int{Val: v}}
	return &m
}
