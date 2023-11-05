package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/errtypes"
)

type Repository interface {
	AddOrUpdate(key string, val string, mtype string) (string, error)
	Delete(key string) error
	Get(key string, mtype string) (metrics.Metric, error)
	GetAll() ([]metrics.Metric, error)
	AddMetricsBulk(metrics []metrics.Metric) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
	HealthCheck() error
	Save(w io.Writer) error
	Load(reader io.Reader) error
	Close() error
}

type CommonMetricsRepository struct {
	storage map[string]metrics.Metric
	lock    sync.RWMutex
}

func NewCommonMetricsRepository() *CommonMetricsRepository {
	r := CommonMetricsRepository{storage: make(map[string]metrics.Metric)}
	return &r
}

func (r *CommonMetricsRepository) HealthCheck() error {
	return nil
}

func (r *CommonMetricsRepository) GetAll() ([]metrics.Metric, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	res := make([]metrics.Metric, 0, len(r.storage))
	for _, m := range r.storage {
		res = append(res, m)
	}

	return res, nil
}

func (r *CommonMetricsRepository) Close() error {
	return nil
}

func (r *CommonMetricsRepository) AddMetricsBulk(metrics []metrics.Metric) error {
	return errors.New("not implemented")
}

func (r *CommonMetricsRepository) AddOrUpdate(key string, val string, mtype string) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !metrics.IsValidMetricType(mtype) {
		return "", errtypes.MakeBadDataError(fmt.Errorf("invalid metric type %v", mtype))
	}

	m, ok := r.storage[key]
	if !ok {
		m, err := metrics.NewMetric(key, val, mtype)
		if err != nil {
			return "", errtypes.MakeBadDataError(err)
		}
		r.storage[key] = m
		return val, nil
	}

	err := m.UpdateData(val)

	if err != nil {
		return "", errtypes.MakeBadDataError(err)
	}

	r.storage[key] = m

	val, _ = m.GetData()

	return val, nil
}

func (r *CommonMetricsRepository) MarshalJSON() ([]byte, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	allMetrics := make([]metrics.Metric, 0, len(r.storage))
	for _, m := range r.storage {
		allMetrics = append(allMetrics, m)
	}
	return json.Marshal(allMetrics)
}

func (r *CommonMetricsRepository) UnmarshalJSON(data []byte) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	allMetrics := make([]metrics.Metric, 0)
	err := json.Unmarshal(data, &allMetrics)

	if err != nil {
		return err
	}

	for _, m := range allMetrics {
		r.storage[m.ID] = m
	}

	return nil
}

func (r *CommonMetricsRepository) Delete(key string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.storage, key)
	return nil
}

func (r *CommonMetricsRepository) Get(key string, mtype string) (metrics.Metric, error) {
	if !metrics.IsValidMetricType(mtype) {
		return metrics.Metric{}, errtypes.MakeBadDataError(fmt.Errorf("invalid metric type '%v'", mtype))
	}
	r.lock.RLock()
	defer r.lock.RUnlock()
	m, ok := r.storage[key]
	if !ok || m.MType != mtype {
		return metrics.Metric{}, errtypes.MakeNotFoundError(fmt.Errorf("metric '%v' not found", key))
	}

	return m, nil
}

func (r *CommonMetricsRepository) Save(w io.Writer) error {
	return json.NewEncoder(w).Encode(&r)
}

func (r *CommonMetricsRepository) Load(reader io.Reader) error {
	b := bytes.Buffer{}
	io.Copy(&b, reader)

	return json.Unmarshal(b.Bytes(), &r)
}
