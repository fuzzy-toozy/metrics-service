package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/errtypes"
)

type Repository interface {
	AddOrUpdate(key string, val string, mtype string) (string, error)
	Delete(key string) error
	Get(key string, mtype string) (metrics.Metric, error)
	GetAll() ([]metrics.Metric, error)
	AddMetricsBulk(metrics []metrics.Metric) ([]metrics.Metric, error)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
	HealthCheck() error
	Save(w io.Writer) error
	Load(reader io.Reader) error
	Close() error
}

type CommonMetricsRepository struct {
	storage      map[string]metrics.Metric
	storageSaver io.Writer
	log          logging.Logger
	lock         sync.RWMutex
}

func NewCommonMetricsRepository(storageSaver io.Writer, log logging.Logger) *CommonMetricsRepository {
	return &CommonMetricsRepository{storage: make(map[string]metrics.Metric),
		log:          log,
		storageSaver: storageSaver}
}

func (r *CommonMetricsRepository) saveData() {
	if r.storageSaver == nil {
		return
	}

	err := r.Save(r.storageSaver)

	if err != nil {
		r.log.Errorf("Failed to save metrics to persistent storage: %v", err)
	}
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

func (r *CommonMetricsRepository) AddMetricsBulk(metricsData []metrics.Metric) ([]metrics.Metric, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	res := make([]metrics.Metric, len(metricsData))
	for i, m := range metricsData {
		val, err := m.GetData()
		if err != nil {
			return nil, err
		}
		uVal, err := r.addOrUpdateUnsafe(m.ID, val, m.MType)
		if err != nil {
			return nil, err
		}

		updatedMetric, err := metrics.NewMetric(m.ID, uVal, m.MType)
		if err != nil {
			return nil, err
		}

		res[i] = updatedMetric
	}

	r.saveData()

	return res, nil
}

func (r *CommonMetricsRepository) addOrUpdateUnsafe(key string, val string, mtype string) (string, error) {
	if !metrics.IsValidMetricType(mtype) {
		return "", errtypes.MakeBadDataError(fmt.Errorf("invalid metric type %v", mtype))
	}

	m, ok := r.storage[key]
	if !ok {
		var err error
		m, err = metrics.NewMetric(key, val, mtype)
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

	val, err = m.GetData()

	if err != nil {
		return "", err
	}

	return val, nil
}

func (r *CommonMetricsRepository) AddOrUpdate(key string, val string, mtype string) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	res, err := r.addOrUpdateUnsafe(key, val, mtype)

	if err != nil {
		r.saveData()
	}

	return res, err
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
	r.saveData()

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
	_, err := io.Copy(&b, reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(b.Bytes(), &r)
}
