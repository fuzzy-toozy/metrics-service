package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/fuzzy-toozy/metrics-service/internal/server/routing"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"github.com/stretchr/testify/require"
)

type RespChecker struct {
	expect common.MetricJSON
	t      *testing.T
}

func (r *RespChecker) Check(req *httptest.ResponseRecorder) {
	data := common.MetricJSON{}
	err := json.NewDecoder(req.Body).Decode(&data)
	require.NoError(r.t, err)
	require.Equal(r.t, data.ID, r.expect.ID)
	require.Equal(r.t, data.MType, r.expect.MType)
	d1, err := data.GetData()
	require.NoError(r.t, err)
	d2, err := r.expect.GetData()
	require.NoError(r.t, err)
	require.Equal(r.t, d1, d2)
}

func TestMetricRegistryHandler_ServeHTTP(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}

	makeMetric := func(mname, mtype, mvalue string) common.MetricJSON {
		data := common.MetricJSON{ID: mname, MType: mtype}
		require.NoError(t, data.SetData(mvalue))
		return data
	}

	makeJSONRequest := func(uri string, data common.MetricJSON) *http.Request {
		var buffer bytes.Buffer
		require.NoError(t, json.NewEncoder(&buffer).Encode(data))
		req, err := http.NewRequest(http.MethodPost, uri, &buffer)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	tests := []struct {
		name        string
		args        args
		wantCode    int
		respChecker *RespChecker
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
		{name: "Set counter metric",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/counter/one/999", nil)},
			wantCode: http.StatusOK},
		{name: "Get counter metric",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/value/counter/one", nil)},
			wantCode: http.StatusOK},
		{name: "Set gauge metric",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodPost, "/update/gauge/one/99.99", nil)},
			wantCode: http.StatusOK},
		{name: "Get gauge metric",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/value/gauge/one", nil)},
			wantCode: http.StatusOK},
		{name: "Get all metrics",
			args:     args{w: httptest.NewRecorder(), r: httptest.NewRequest(http.MethodGet, "/", nil)},
			wantCode: http.StatusOK},
		{name: "Set gauge metric JSON",
			args: args{w: httptest.NewRecorder(), r: makeJSONRequest("/update",
				makeMetric("one", common.MetricTypeGauge, "10.999"))},
			wantCode:    http.StatusOK,
			respChecker: &RespChecker{t: t, expect: makeMetric("one", common.MetricTypeGauge, "10.999")},
		},
		{name: "Get gauge metric JSON",
			args: args{w: httptest.NewRecorder(), r: makeJSONRequest("/value",
				makeMetric("one", common.MetricTypeGauge, "10.999"))},
			wantCode:    http.StatusOK,
			respChecker: &RespChecker{t: t, expect: makeMetric("one", common.MetricTypeGauge, "10.999")},
		},
		{name: "Set counter metric JSON",
			args: args{w: httptest.NewRecorder(), r: makeJSONRequest("/update",
				makeMetric("one", common.MetricTypeGauge, "10.999"))},
			wantCode:    http.StatusOK,
			respChecker: &RespChecker{t: t, expect: makeMetric("one", common.MetricTypeGauge, "10.999")},
		},
		{name: "Get counter metric JSON",
			args: args{w: httptest.NewRecorder(), r: makeJSONRequest("/value",
				makeMetric("one", common.MetricTypeCounter, "999"))},
			wantCode:    http.StatusOK,
			respChecker: &RespChecker{t: t, expect: makeMetric("one", common.MetricTypeCounter, "999")},
		},
	}

	registry := storage.NewCommonMetricsStorage()
	registry.AddRepository("gauge", storage.NewGaugeMetricRepository())
	registry.AddRepository("counter", storage.NewCounterMetricRepository())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewMetricRegistryHandler(registry, log.NewDevZapLogger(),
				handlers.MetricURLInfo{Type: "mtype", Name: "mname", Value: "mval"}, nil, config.DBConfig{})
			r := routing.SetupRouting(h)
			r.ServeHTTP(tt.args.w, tt.args.r)
			resp := tt.args.w.(*httptest.ResponseRecorder)
			require.Equal(t, tt.wantCode, resp.Code)

			if tt.respChecker != nil {
				tt.respChecker.Check(resp)
			}
		})
	}
}
