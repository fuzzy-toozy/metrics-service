package storage

import "github.com/fuzzy-toozy/metrics-service/internal/metrics"

type MetricsStorage interface {
	Clear()
	AddOrUpdate(m metrics.Metric) error
	GetAllMetrics() []metrics.Metric
}

type CommonMetricsStorage struct {
	storage []metrics.Metric
}

func (s *CommonMetricsStorage) AddOrUpdate(m metrics.Metric) error {
	s.storage = append(s.storage, m)
	return nil
}

func (s *CommonMetricsStorage) GetAllMetrics() []metrics.Metric {
	return s.storage
}

func (s *CommonMetricsStorage) Clear() {
	s.storage = s.storage[:0]
}

func NewCommonMetricsStorage() *CommonMetricsStorage {
	s := CommonMetricsStorage{}
	return &s
}
