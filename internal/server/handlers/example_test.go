package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
)

var serverWg sync.WaitGroup
var serverApp *http.Server

func setupServer() error {
	logger := log.NewDevZapLogger()
	h := NewMetricRegistryHandler(storage.NewCommonMetricsRepository(),
		logger, MetricURLInfo{
			Name:  "metricName",
			Value: "metricValue",
			Type:  "metricType",
		}, nil, config.DBConfig{})

	handler := SetupRouting(h)

	serverApp = &http.Server{
		Addr:    "localhost:8090",
		Handler: handler,
	}

	serverWg.Add(1)
	go func() {
		err := serverApp.ListenAndServe()
		if err != nil {
			logger.Infof("Server stopped. Reason: %v", err)
		}
		defer serverWg.Done()
	}()

	var err error
	retryCnt := 5
	for i := 0; i < retryCnt; i++ {
		var conn net.Conn
		conn, err = net.DialTimeout("tcp", net.JoinHostPort("localhost", "8090"), 1*time.Second)
		if err == nil {
			conn.Close()
			break
		}
	}

	if err != nil {
		return fmt.Errorf("server took too long time to setup: %v", err)
	}

	return nil
}

func tearDownServer() {
	serverApp.Shutdown(context.Background())
	serverWg.Wait()
}

func ExampleMetricRegistryHandler_UpdateMetricFromJSON() {
	err := setupServer()
	if err != nil {
		fmt.Printf("Failed to setup server: %v", err)
		return
	}
	defer tearDownServer()

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	const serverURL = "http://localhost:8090"
	const someMetric = "{\"id\": \"metricId\", \"type\": \"gauge\", \"value\": 11.22}"

	req, err := http.NewRequest(http.MethodPost, serverURL+"/update", bytes.NewBuffer([]byte(someMetric)))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	respBuf, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	defer resp.Body.Close()

	fmt.Println(string(respBuf))

	// Output: {"id":"metricId","type":"gauge","value":11.22}
}

func ExampleMetricRegistryHandler_UpdateMetricsFromJSON() {
	err := setupServer()
	if err != nil {
		fmt.Printf("Failed to setup server: %v", err)
		return
	}
	defer tearDownServer()

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	const serverURL = "http://localhost:8090"
	const someMetrics = "[{\"id\": \"metricId1\", \"type\": \"gauge\", \"value\": 11.22}, {\"id\": \"metricId2\", \"type\": \"counter\", \"delta\": 31}]"

	req, err := http.NewRequest(http.MethodPost, serverURL+"/updates", bytes.NewBuffer([]byte(someMetrics)))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	respBuf, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	defer resp.Body.Close()

	fmt.Println(string(respBuf))

	// Output: [{"id":"metricId1","type":"gauge","value":11.22},{"id":"metricId2","type":"counter","delta":31}]
}

func ExampleMetricRegistryHandler_UpdateMetric() {
	err := setupServer()
	if err != nil {
		fmt.Printf("Failed to setup server: %v", err)
		return
	}
	defer tearDownServer()

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	const serverURL = "http://localhost:8090"
	const metricName = "metricId"
	const metricType = "gauge"
	const metricVal = "42.42"

	reqURL := fmt.Sprintf("%v/%v/%v/%v/%v", serverURL, "update", metricType, metricName, metricVal)
	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	respBuf, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	defer resp.Body.Close()

	fmt.Println(string(respBuf))

	// Output: 42.42
}

func ExampleMetricRegistryHandler_GetMetric() {
	err := setupServer()
	if err != nil {
		fmt.Printf("Failed to setup server: %v", err)
		return
	}
	defer tearDownServer()

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	const serverURL = "http://localhost:8090"
	const metricName = "metricId"
	const metricType = "gauge"
	const metricVal = "42.42"

	reqURL := fmt.Sprintf("%v/%v/%v/%v/%v", serverURL, "update", metricType, metricName, metricVal)
	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	resp.Body.Close()

	reqURL = fmt.Sprintf("%v/%v/%v/%v", serverURL, "value", metricType, metricName)

	req, err = http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v", err)
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	defer resp.Body.Close()

	respBuf, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	fmt.Println(string(respBuf))
	// Output: 42.42
}

func ExampleMetricRegistryHandler_GetMetricJSON() {
	err := setupServer()
	if err != nil {
		fmt.Printf("Failed to setup server: %v", err)
		return
	}
	defer tearDownServer()

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	const serverURL = "http://localhost:8090"
	const metricName = "metricId"
	const metricType = "gauge"
	const metricVal = "42.42"

	reqURL := fmt.Sprintf("%v/%v/%v/%v/%v", serverURL, "update", metricType, metricName, metricVal)
	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	resp.Body.Close()

	reqURL = fmt.Sprintf("%v/%v", serverURL, "value")
	reqData := fmt.Sprintf("{ \"id\": \"%v\", \"type\": \"%v\"}", metricName, metricType)

	req, err = http.NewRequest(http.MethodPost, reqURL, bytes.NewBufferString(reqData))
	if err != nil {
		fmt.Printf("Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}

	defer resp.Body.Close()

	respBuf, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read request: %v\n", err)
		return
	}

	fmt.Println(string(respBuf))
	// Output: {"id":"metricId","type":"gauge","value":42.42}
}
