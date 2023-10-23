package metrics

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

func (m *GaugeMetric) GetType() string {
	return common.MetricTypeGauge
}

func (m *GaugeMetric) UpdateValue(v Metric) {
	metric := v.(*GaugeMetric)
	m.Val = metric.Val
}

type CounterMetric struct {
	common.Int
}

func (m *CounterMetric) GetType() string {
	return common.MetricTypeCounter
}

func (m *CounterMetric) UpdateValue(v Metric) {
	metric := v.(*CounterMetric)
	m.Val += metric.Val
}

func NewGaugeMeric(v float64) *GaugeMetric {
	m := GaugeMetric{Float: common.Float{Val: v}}
	return &m
}

func NewCounterMeric(v int64) *CounterMetric {
	m := CounterMetric{Int: common.Int{Val: v}}
	return &m
}
