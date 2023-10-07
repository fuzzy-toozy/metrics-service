package storage

import "github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/metrics"

type MetricsStorage interface {
	AddOrUpdate(name string, m metrics.Metric) error
	Delete(name string) error
	ForEachMetric(callback func(name string, m metrics.Metric) error) error
}

type CommonMetricsStorage struct {
	storage map[string]metrics.Metric
}

func (s *CommonMetricsStorage) AddOrUpdate(name string, m metrics.Metric) error {
	s.storage[name] = m
	return nil
}

func (s *CommonMetricsStorage) Delete(name string) error {
	delete(s.storage, name)
	return nil
}

func (s *CommonMetricsStorage) ForEachMetric(callback func(name string, m metrics.Metric) error) error {
	for n, m := range s.storage {
		err := callback(n, m)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewCommonMetricsStorage() *CommonMetricsStorage {
	s := CommonMetricsStorage{storage: make(map[string]metrics.Metric)}
	return &s
}
