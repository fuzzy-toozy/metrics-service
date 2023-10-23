package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
)

type Metric interface {
	GetValue() string
	GetLastTimeUpdated() time.Time
	SetLastTimeUpdated(t time.Time)
	UpdateValue(v string) error
	MarshalJSON() ([]byte, error)
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

func valToBytes(m Metric) []byte {
	return []byte(fmt.Sprintf("\"%v\"", m.GetValue()))
}

func (m *CounterMetric) MarshalJSON() ([]byte, error) {
	return valToBytes(m), nil
}

func (m *GaugeMetric) MarshalJSON() ([]byte, error) {
	return valToBytes(m), nil
}

func unmarshalJSON(data []byte, m Metric) error {
	row := string(data[:])
	rowTrimmed := strings.Trim(row, "{}")
	vals := strings.Split(rowTrimmed, ":")

	if len(vals) != 2 {
		return fmt.Errorf("incorrect format")
	}

	return m.UpdateValue(strings.Trim(vals[1], "\""))
}

func (m *GaugeMetric) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, m)
}

func (m *CounterMetric) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, m)
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
	AddOrUpdate(key string, val string) (string, error)
	Delete(key string) error
	Get(key string) (Metric, error)
	ForEachMetric(func(name string, m Metric) error) error
	AddMetricsBulk(metrics []common.MetricJSON) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

type CommonMetricRepository struct {
	storage map[string]Metric
	lock    sync.RWMutex
}

func (r *CommonMetricRepository) AddMetricsBulk(metrics []common.MetricJSON) error {
	return errors.New("not implemented")
}

func (r *CommonMetricRepository) MarshalJSON() ([]byte, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return json.Marshal(r.storage)
}

func (r *CommonMetricRepository) UnmarshalJSON(data []byte) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	m := map[string]map[string]string{}
	err := json.Unmarshal(data, &m)
	return err
}

func (r *CommonMetricRepository) ForEachMetric(callback func(name string, m Metric) error) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	for n, m := range r.storage {
		err := callback(n, m)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *CommonMetricRepository) Delete(key string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.storage, key)
	return nil
}

func (r *CommonMetricRepository) Get(key string) (Metric, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	val, ok := r.storage[key]
	if !ok {
		return nil, fmt.Errorf("no metric for key %v", key)
	}

	return val, nil
}

type GaugeMetricRepository struct {
	CommonMetricRepository
}

func (r *GaugeMetricRepository) AddOrUpdate(key string, val string) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
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

func (r *CounterMetricRepository) AddOrUpdate(key string, val string) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return addOrUpdate(key, val, &CounterMetric{}, r.storage)
}

func addOrUpdate(key string, val string, m Metric, storage map[string]Metric) (string, error) {
	v, ok := storage[key]
	if !ok {
		err := m.UpdateValue(val)
		if err != nil {
			return "", err
		}
		m.SetLastTimeUpdated(time.Now())
		storage[key] = m

		return m.GetValue(), nil
	} else {
		err := v.UpdateValue(val)
		if err != nil {
			return "", err
		}

		return v.GetValue(), nil
	}
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
	ForEachRepository(func(name string, repo Repository) error) error
	Save(w io.Writer) error
	Load(w io.Reader) error
}

type CommonMetricsStorage struct {
	storage map[string]Repository
}

func (s *CommonMetricsStorage) Save(w io.Writer) error {
	return json.NewEncoder(w).Encode(s.storage)
}

func (s *CommonMetricsStorage) Load(r io.Reader) error {
	m := map[string]map[string]string{}
	b := bytes.Buffer{}
	io.Copy(&b, r)

	err := json.Unmarshal(b.Bytes(), &m)
	if err != nil {
		return err
	}

	for storageName, metricsStorage := range m {
		repo, err := s.GetRepository(storageName)
		if err != nil {
			return err
		}
		for metricName, metricValue := range metricsStorage {
			_, err := repo.AddOrUpdate(metricName, metricValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *CommonMetricsStorage) ForEachRepository(callback func(name string, repo Repository) error) error {
	for n, r := range s.storage {
		err := callback(n, r)

		if err != nil {
			return err
		}
	}

	return nil
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
	storage.AddRepository("gauge", NewGaugeMetricRepository())
	storage.AddRepository("counter", NewCounterMetricRepository())
	return &storage
}
