package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/storage"
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
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/invalid/10", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Invalid metric value gauge string",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/ff", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Invalid metric value counter string",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/ff", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Invalid metric value counter float",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/10.11", nil)},
			wantCode: http.StatusBadRequest},
		{name: "Valid gauge",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/10.11", nil)},
			wantCode: http.StatusOK},
		{name: "Valid counter",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/10", nil)},
			wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := storage.NewCommonMetricsStorage()
			registry.AddRepository("gauge", storage.NewGaugeMetricRepository())
			registry.AddRepository("counter", storage.NewCounterMetricRepository())
			h := NewMetricRegistryHandler(registry, log.NewDevZapLogger())
			h.ServeHTTP(tt.args.w, tt.args.r)
		})
	}
}
