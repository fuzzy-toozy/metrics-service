package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestMetricRegistryHandler_ServeHTTP(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name     string
		args     args
		wantCode int
	}{
		{name: "Invalid path",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/invalid", nil)},
			wantCode: http.StatusNotFound},
		{name: "Invalid metric type",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/invalid/name/10", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Invalid metric value gauge string",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/name/ff", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Invalid metric value counter string",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/name/ff", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Invalid metric value counter float",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/name/10.11", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Valid gauge",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/name/10.11", nil)},
			wantCode: http.StatusOK},
		{name: "Valid counter",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/name/10", nil)},
			wantCode: http.StatusOK},
		{name: "Get counter metric",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/value/counter/one", nil)},
			wantCode: http.StatusOK},
		{name: "Get gauge metric",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/value/gauge/one", nil)},
			wantCode: http.StatusOK},
		{name: "Get all metrics",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/", nil)},
			wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := storage.NewCommonMetricsStorage()
			registry.AddRepository("gauge", storage.NewGaugeMetricRepository())
			registry.AddRepository("counter", storage.NewCounterMetricRepository())
			repo, err := registry.GetRepository("gauge")
			require.NoError(t, err)
			err = repo.AddOrUpdate("one", "10.10")
			require.NoError(t, err)
			repo, err = registry.GetRepository("counter")
			require.NoError(t, err)
			err = repo.AddOrUpdate("one", "1000")
			require.NoError(t, err)

			h := NewMetricRegistryHandler(registry, log.NewDevZapLogger(), MetricURLInfo{Type: "mtype", Name: "mname", Value: "mval"})
			r := SetupRouting(h)
			r.ServeHTTP(tt.args.w, tt.args.r)
			require.Equal(t, tt.wantCode, tt.args.w.(*httptest.ResponseRecorder).Code)
		})
	}
}
