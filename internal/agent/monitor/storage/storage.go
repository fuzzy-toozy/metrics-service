package storage

import (
	"encoding/json"

	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
)

type MetricsStorage interface {
	Clear()
	AddOrUpdate(m metrics.Metric) error
	GetAllMetrics() StorageMetrics
}

type StorageMetric metrics.Metric

func (m StorageMetric) MarshalJSON() ([]byte, error) {
	return json.Marshal(metrics.Metric(m))
}

type StorageMetrics []metrics.Metric

func (m StorageMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal([]metrics.Metric(m))
}

type CommonMetricsStorage struct {
	storage []StorageMetric
}

func (s *CommonMetricsStorage) AddOrUpdate(m metrics.Metric) error {
	s.storage = append(s.storage, StorageMetric(m))
	return nil
}

func (s *CommonMetricsStorage) GetAllMetrics() StorageMetrics {
	cp := make([]metrics.Metric, len(s.storage))
	for i, m := range s.storage {
		cp[i] = metrics.Metric(m)
	}
	return cp
}

func (s *CommonMetricsStorage) Clear() {
	s.storage = s.storage[:0]
}

func NewCommonMetricsStorage() *CommonMetricsStorage {
	s := CommonMetricsStorage{}
	return &s
}
