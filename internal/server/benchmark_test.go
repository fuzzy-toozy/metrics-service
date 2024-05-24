package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/beevik/guid"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/fuzzy-toozy/metrics-service/internal/server/service"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
)

func makeJSONMetricReq(m metrics.Metric) []byte {
	buffer := bytes.NewBuffer(nil)
	m.Delta = nil
	m.Value = nil
	err := json.NewEncoder(buffer).Encode(m)
	if err != nil {
		panic(fmt.Sprintf("failed to encode metric %v", err))
	}
	return buffer.Bytes()
}

func makeRandomMetric() metrics.Metric {
	randVal := rand.Int31() % 2
	if randVal == 0 {
		return metrics.NewGaugeMetric(guid.NewString(), rand.Float64())
	} else {
		return metrics.NewCounterMetric(guid.NewString(), int64(rand.Uint64()))
	}
}

func makeRandomMetricJSON() []byte {
	m := makeRandomMetric()
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(m)
	if err != nil {
		panic(fmt.Sprintf("failed to encode metric %v", err))
	}
	return buf.Bytes()
}

func getMetricValPath(m metrics.Metric) string {
	val, err := m.GetData()
	if err != nil {
		panic(fmt.Sprintf("Failed to get meric data: %v", err))
	}
	return fmt.Sprintf("/%v/%v/%v", m.MType, m.ID, val)
}

func getMetricPath(m metrics.Metric) string {
	return fmt.Sprintf("/%v/%v", m.MType, m.ID)
}

func makeRandomGaugeMetricPath() string {
	m := metrics.NewGaugeMetric(guid.NewString(), rand.Float64())
	return getMetricValPath(m)
}

func makeRandomCounterMetricPath() string {
	m := metrics.NewCounterMetric(guid.NewString(), int64(rand.Uint64()))
	val, err := m.GetData()
	if err != nil {
		panic(fmt.Sprintf("Failed to get meric data: %v", err))
	}
	return fmt.Sprintf("/%v/%v/%v", m.MType, m.ID, val)
}

func makeMetricsJSON(m []metrics.Metric) []byte {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(m)
	if err != nil {
		panic(fmt.Sprintf("failed to encode metric %v", err))
	}
	return buf.Bytes()
}

type rewindHandler struct {
	buf     *bytes.Reader
	handler http.Handler
	r       *http.Request
}

type simpleHandler struct {
	r       *http.Request
	handler http.Handler
}

func (h *simpleHandler) ServeHTTP(w http.ResponseWriter) {
	h.handler.ServeHTTP(w, h.r)
}

func newSimpleHandler(h http.Handler, url string, method string, contentType string, data []byte) *simpleHandler {
	res := &simpleHandler{
		r:       httptest.NewRequest(method, url, bytes.NewBuffer(data)),
		handler: h,
	}

	if len(contentType) > 0 {
		res.r.Header.Set("Content-Type", contentType)
	}

	return res
}

func (h *rewindHandler) ServeHTTP(w http.ResponseWriter) {
	h.handler.ServeHTTP(w, h.r)
	_, err := h.buf.Seek(0, 0)
	if err != nil {
		panic(fmt.Sprintf("Failed to seek buffer: %v", err))
	}
}

func newRewindHandler(h http.Handler, url string, method string, contentType string, data []byte) *rewindHandler {
	rh := &rewindHandler{}
	buf := bytes.NewReader(data)
	r := httptest.NewRequest(method, url, buf)

	rh.buf = buf
	rh.r = r
	rh.handler = h

	if len(contentType) > 0 {
		rh.r.Header.Set("Content-Type", contentType)
	}

	return rh
}

func BenchmarkHandlers(b *testing.B) {
	const metricsNum = 100

	registry := storage.NewCommonMetricsRepository()
	h := handlers.NewMetricRegistryHandler(service.NewCommonMetricsServiceHTTP(registry), log.NewDevZapLogger(),
		handlers.MetricURLInfo{Type: "mtype", Name: "mname", Value: "mval"}, nil)
	serverHandler := handlers.SetupRouting(h)

	randomMetrics := make([]metrics.Metric, 0, metricsNum)
	jsonBulkHandlers := make([]*rewindHandler, metricsNum)

	for i := range jsonBulkHandlers {
		randomMetrics = randomMetrics[:0]
		for j := 0; j < metricsNum; j++ {
			randomMetrics = append(randomMetrics, makeRandomMetric())
		}
		jsonBulkHandlers[i] = newRewindHandler(serverHandler, "/updates", http.MethodPost, "application/json", makeMetricsJSON(randomMetrics))
	}

	// put metrics to storage so we know they are there (for get tests
	testH := newRewindHandler(serverHandler, "/updates", http.MethodPost, "application/json", makeMetricsJSON(randomMetrics))
	testH.ServeHTTP(httptest.NewRecorder())

	gaugeHandlers := make([]*simpleHandler, metricsNum)
	for i := range gaugeHandlers {
		gaugeHandlers[i] = newSimpleHandler(serverHandler, "/update"+makeRandomGaugeMetricPath(), http.MethodPost, "", nil)
	}

	counterHandlers := make([]*simpleHandler, metricsNum)
	for i := range counterHandlers {
		counterHandlers[i] = newSimpleHandler(serverHandler, "/update"+makeRandomCounterMetricPath(), http.MethodPost, "", nil)
	}

	jsonHandlers := make([]*rewindHandler, metricsNum)
	for i := range jsonHandlers {
		jsonHandlers[i] = newRewindHandler(serverHandler, "/update", http.MethodPost, "application/json", makeRandomMetricJSON())
	}

	b.ResetTimer()

	b.Run("Gauge metric update", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			res := httptest.NewRecorder()
			gaugeHandlers[rand.Int()%metricsNum].ServeHTTP(res)
			resp := res.Result()
			status := resp.StatusCode
			err := resp.Body.Close()
			runtime.KeepAlive(err)

			if status != http.StatusOK {
				fmt.Printf("Failed request %v\n", status)
				return
			}
		}
	})

	b.Run("Counter metric update", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			res := httptest.NewRecorder()
			counterHandlers[rand.Int()%metricsNum].ServeHTTP(res)
			resp := res.Result()
			status := resp.StatusCode
			err := resp.Body.Close()
			runtime.KeepAlive(err)

			if status != http.StatusOK {
				fmt.Printf("Failed request %v\n", status)
				return
			}
		}
	})

	b.Run("JSON metric update", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			res := httptest.NewRecorder()
			jsonHandlers[rand.Int()%metricsNum].ServeHTTP(res)
			resp := res.Result()
			status := resp.StatusCode
			err := resp.Body.Close()
			runtime.KeepAlive(err)

			if status != http.StatusOK {
				fmt.Printf("Failed request %v\n", status)
				return
			}
		}
	})

	b.Run("Bulk JSON metrics update", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			res := httptest.NewRecorder()
			jsonBulkHandlers[rand.Int()%metricsNum].ServeHTTP(res)
			resp := res.Result()
			status := resp.StatusCode
			err := resp.Body.Close()
			runtime.KeepAlive(err)

			if status != http.StatusOK {
				fmt.Printf("Failed request %v\n", status)
				return
			}
		}
	})

	b.StopTimer()

	getValTests := make([]*http.Request, metricsNum)
	for i, m := range randomMetrics {
		getValTests[i] = httptest.NewRequest(http.MethodGet, "/value"+getMetricPath(m), nil)
	}

	b.StartTimer()
	b.Run("Get metric value", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			res := httptest.NewRecorder()

			serverHandler.ServeHTTP(res, getValTests[rand.Int()%metricsNum])
			resp := res.Result()
			status := resp.StatusCode
			err := resp.Body.Close()
			runtime.KeepAlive(err)

			if status != http.StatusOK {
				fmt.Printf("Failed reques %v\n", status)
				return
			}
		}
	})
	b.StopTimer()

	getValJSONTests := make([]*rewindHandler, metricsNum)
	for i, m := range randomMetrics {
		getValJSONTests[i] = newRewindHandler(serverHandler, "/value", http.MethodPost, "application/json", makeJSONMetricReq(m))
	}

	b.StartTimer()
	b.Run("Get metric value JSON", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			res := httptest.NewRecorder()

			getValJSONTests[rand.Int()%metricsNum].ServeHTTP(res)
			resp := res.Result()
			status := resp.StatusCode
			err := resp.Body.Close()
			runtime.KeepAlive(err)

			if status != http.StatusOK {
				fmt.Printf("Failed reques %v\n", status)
				return
			}
		}
	})
}
