package service

import (
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/errtypes"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
)

type MetricsService interface {
	GetMetric(name string, mtype string) (metrics.Metric, ServiceError)
	UpdateMetric(mtype, mname, mvalue string) (metrics.Metric, ServiceError)
	GetAllMetrics() ([]metrics.Metric, ServiceError)
	UpdateMetrics(metrics []metrics.Metric) ServiceError
	HealthCheck() error
}

type ServiceError interface {
	Error() string
	Unwrap() error
	Code() int
}

type CommonServiceError struct {
	code int
	errtypes.GenericErrorWrapper
}

func (e CommonServiceError) Code() int {
	return e.code
}

func MakeServiceError(code int, err error) CommonServiceError {
	return CommonServiceError{
		code:                code,
		GenericErrorWrapper: errtypes.MakeGenericErrorWrapper(err),
	}
}

type CommonMetricsService struct {
	registry      storage.Repository
	errorToStatus func(err error) int
}

func NewCommonMetricsService(registry storage.Repository, errToStatus func(err error) int) *CommonMetricsService {
	return &CommonMetricsService{
		registry:      registry,
		errorToStatus: errToStatus,
	}
}

func NewCommonMetricsServiceHTTP(registry storage.Repository) *CommonMetricsService {
	return &CommonMetricsService{
		registry:      registry,
		errorToStatus: errtypes.ErrorToStatusHTTP,
	}
}

func NewCommonMetricsServiceGRPC(registry storage.Repository) *CommonMetricsService {
	return &CommonMetricsService{
		registry:      registry,
		errorToStatus: errtypes.ErrorToStatusGRPC,
	}
}

var _ MetricsService = (*CommonMetricsService)(nil)

func (s *CommonMetricsService) HealthCheck() error {
	err := s.registry.HealthCheck()
	if err != nil {
		return MakeServiceError(s.errorToStatus(err), err)
	}
	return nil
}

func (s *CommonMetricsService) GetMetric(name string, mtype string) (metrics.Metric, ServiceError) {
	m, err := s.registry.Get(name, mtype)

	if err != nil {
		return m, MakeServiceError(s.errorToStatus(err), err)
	}

	return m, nil
}

func (s *CommonMetricsService) UpdateMetric(mtype, mname, mvalue string) (metrics.Metric, ServiceError) {
	updatedVal, err := s.registry.AddOrUpdate(mname, mvalue, mtype)

	if err != nil {
		return metrics.Metric{}, MakeServiceError(s.errorToStatus(err), err)
	}

	m, err := metrics.NewMetric(mname, updatedVal, mtype)
	if err != nil {
		return metrics.Metric{}, MakeServiceError(s.errorToStatus(err), err)
	}

	return m, nil
}

func (s *CommonMetricsService) GetAllMetrics() ([]metrics.Metric, ServiceError) {
	repoMetrics, err := s.registry.GetAll()

	if err != nil {
		return nil, MakeServiceError(s.errorToStatus(err), err)
	}

	return repoMetrics, nil
}

func (s *CommonMetricsService) UpdateMetrics(metrics []metrics.Metric) ServiceError {
	err := s.registry.AddMetricsBulk(metrics)
	if err != nil {
		return MakeServiceError(s.errorToStatus(err), err)
	}

	return nil
}
