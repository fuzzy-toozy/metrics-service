package storage

import (
	"fmt"
	"strconv"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
)

type Metric interface {
	GetValue() string
	GetLastTimeUpdated() time.Time
	SetLastTimeUpdated(t time.Time)
	UpdateValue(v string) error
}

type MetricUpdateTime struct {
	LastTimeUpdated time.Time
}

func (m MetricUpdateTime) GetLastTimeUpdated() time.Time {
	return m.LastTimeUpdated
}

func (m *MetricUpdateTime) SetLastTimeUpdated(t time.Time) {
	m.LastTimeUpdated = t
}

type GaugeMetric struct {
	common.Float
	MetricUpdateTime
}

type CounterMetric struct {
	common.Int
	MetricUpdateTime
}

func (m *GaugeMetric) UpdateValue(v string) error {
	val, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return err
	}
	m.Val = val
	return nil
}

func (m *CounterMetric) UpdateValue(v string) error {
	val, err := strconv.ParseInt(v, 10, 64)

	if err != nil {
		return err
	}

	m.Val += val
	return nil
}

type Repository interface {
	AddOrUpdate(key string, val string) error
	Delete(key string) error
	Get(key string) (Metric, error)
}

type CommonMetricRepository struct {
	storage map[string]Metric
}

func (r *CommonMetricRepository) Delete(key string) error {
	delete(r.storage, key)
	return nil
}

func (r *CommonMetricRepository) Get(key string) (Metric, error) {
	val, ok := r.storage[key]
	if !ok {
		return nil, fmt.Errorf("no metric for key %v", key)
	}

	return val, nil
}

type GaugeMetricRepository struct {
	CommonMetricRepository
}

func (r *GaugeMetricRepository) AddOrUpdate(key string, val string) error {
	return addOrUpdate(key, val, &GaugeMetric{}, r.storage)

}

func NewGaugeMetricRepository() *GaugeMetricRepository {
	repo := GaugeMetricRepository{}
	repo.storage = make(map[string]Metric)
	return &repo
}

type CounterMetricRepository struct {
	CommonMetricRepository
}

func (r *CounterMetricRepository) AddOrUpdate(key string, val string) error {
	return addOrUpdate(key, val, &CounterMetric{}, r.storage)
}

func addOrUpdate(key string, val string, m Metric, storage map[string]Metric) error {
	v, ok := storage[key]
	if !ok {
		err := m.UpdateValue(val)
		if err != nil {
			return err
		}
		m.SetLastTimeUpdated(time.Now())
		storage[key] = m
	} else {
		err := v.UpdateValue(val)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewCounterMetricRepository() *CounterMetricRepository {
	repo := CounterMetricRepository{}
	repo.storage = make(map[string]Metric)
	return &repo
}

type MetricsStorage interface {
	GetRepository(name string) (Repository, error)
	AddRepository(name string, repo Repository) error
	DeleteRepository(name string) error
}

type CommonMetricsStorage struct {
	storage map[string]Repository
}

func (s *CommonMetricsStorage) GetRepository(name string) (Repository, error) {
	repo, ok := s.storage[name]
	if !ok {
		return nil, fmt.Errorf("repository '%v' doesn't exist", name)
	}

	return repo, nil
}

func (s *CommonMetricsStorage) AddRepository(name string, repo Repository) error {
	s.storage[name] = repo
	return nil
}

func (s *CommonMetricsStorage) DeleteRepository(name string) error {
	delete(s.storage, name)
	return nil
}

func NewCommonMetricsStorage() *CommonMetricsStorage {
	storage := CommonMetricsStorage{storage: make(map[string]Repository)}
	return &storage
}
