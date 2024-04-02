// Metrics gathering agent.
// Gathers various memory and CPU metrics from local machine
// and sends them to speficied server.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	monitorHttp "github.com/fuzzy-toozy/metrics-service/internal/agent/http"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/compression"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type Agent struct {
	monitors []monitor.Monitor
	log      log.Logger
	config   config.Config
}

type configOption func(agent *Agent)

// WithPsMonitor option to create agent with PsMontior
func WithPsMonitor(a *Agent) {
	a.monitors = append(a.monitors, monitor.NewPsMonitor(storage.NewCommonMetricsStorage(), a.log))
}

// WithCommonMonitor option to create agent with CommonMonitor
func WithCommonMonitor(a *Agent) {
	a.monitors = append(a.monitors, monitor.NewMetricsMonitor(storage.NewCommonMetricsStorage(), a.log))
}

type buffers struct {
	compression bytes.Buffer
	data        bytes.Buffer
}

type worker struct {
	buffs      buffers
	httpClient monitorHttp.HTTPClient
	log        log.Logger
	config     *config.Config
}

type dataType int

const (
	tBULK dataType = iota
	tSINGLE
)

type reportData struct {
	dType dataType
	data  json.Marshaler
}

func NewAgent(config config.Config, logger log.Logger, opts ...configOption) (*Agent, error) {
	a := Agent{config: config,
		log: logger}

	for _, opt := range opts {
		opt(&a)
	}

	if len(a.monitors) == 0 {
		return nil, fmt.Errorf("can't create agent without monitors")
	}

	return &a, nil
}

func newWorker(config *config.Config, logger log.Logger) *worker {
	w := worker{httpClient: monitorHttp.NewDefaultHTTPClient(), log: logger, config: config}
	b := buffers{data: bytes.Buffer{}, compression: bytes.Buffer{}}
	w.buffs = b
	return &w
}

func (w *worker) getCompressedBytes(data []byte) ([]byte, error) {
	if len(w.config.CompressAlgo) > 0 {
		factory, err := compression.GetCompressorFactory(w.config.CompressAlgo)
		if err != nil {
			return nil, fmt.Errorf("failed to get compression factory: %w", err)
		}
		w.buffs.compression.Reset()
		compressor, err := factory(&w.buffs.compression)
		if err != nil {
			return nil, fmt.Errorf("failed to create compressor: %w", err)
		}

		_, err = compressor.Write(w.buffs.data.Bytes())

		if err != nil {
			return nil, fmt.Errorf("failed to compress data: %w", err)
		}

		err = compressor.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to finalize compressor: %w", err)
		}

		return w.buffs.compression.Bytes(), nil
	}

	return nil, fmt.Errorf("no compression algo specified")
}

func (w *worker) reportDataJSON(data reportData) error {
	var reportURL string
	if data.dType == tBULK {
		reportURL = w.config.ReportBulkEndpoint
	} else if data.dType == tSINGLE {
		reportURL = w.config.ReportEndpoint
	} else {
		return fmt.Errorf("wrong report data type %v", data.dType)
	}

	contentType := "application/json"
	contentEncoding := ""

	w.buffs.data.Reset()
	if err := json.NewEncoder(&w.buffs.data).Encode(data.data); err != nil {
		return fmt.Errorf("failed to encode metrics to JSON: %w", err)
	}

	bytesToSend := w.buffs.data.Bytes()

	var sigHash string
	if w.config.SecretKey != nil {
		hash, err := encryption.SignData(bytesToSend, w.config.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to sign request data: %w", err)
		}
		sigHash = hash
	}

	compressedBytes, err := w.getCompressedBytes(bytesToSend)
	if err != nil {
		w.log.Debugf("Unable to enable compression: %v", err)
	} else {
		bytesToSend = compressedBytes
		contentEncoding = w.config.CompressAlgo
	}

	req, err := http.NewRequest(http.MethodPost, reportURL, bytes.NewBuffer(bytesToSend))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Encoding", contentEncoding)

	if len(sigHash) != 0 {
		req.Header.Set("HashSHA256", sigHash)
	}

	resp, err := w.httpClient.Send(req)

	if resp != nil {
		defer func() {
			_, err := io.Copy(io.Discard, resp.Body)
			if err != nil {
				w.log.Debugf("Failed reading request body: %v", err)
			}
			resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("failed to send metrics. Status code: %v", resp.StatusCode)
		}
	}

	return err
}

func (a *Agent) reportMetrics(ctx context.Context, mstorage storage.MetricsStorage, gatherChan chan<- reportData) {
	allMetrics := mstorage.GetAllMetrics()

	if len(allMetrics) == 0 {
		return
	}

	rData := reportData{}
	rData.data = allMetrics
	rData.dType = tBULK

	select {
	case gatherChan <- rData:
	case <-ctx.Done():
		return
	}

	for _, m := range allMetrics {
		rData := reportData{dType: tSINGLE, data: storage.StorageMetric(m)}
		select {
		case gatherChan <- rData:
		case <-ctx.Done():
			return
		}
	}
}

// Run starts agent's metric gathering with all configured monitors.
// Also starts report thread to send gathererd data to server.
func (a *Agent) Run() {
	a.config.Print(a.log)

	gatherChan := make(chan reportData, a.config.RateLimit)
	defer close(gatherChan)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-c
		cancel()
		a.log.Infof("Agent is stopping...")
	}()

	wg := sync.WaitGroup{}

	for _, mon := range a.monitors {
		wg.Add(1)
		currentMonitor := mon
		go func() {
			defer wg.Done()
			reportTicker := time.NewTicker(a.config.ReportInterval)
			for {
				select {
				case <-time.After(a.config.PollInterval):
					err := currentMonitor.GatherMetrics()
					if err != nil {
						a.log.Warnf("Failed to gather app metrics. %v", err)
					}
				case <-reportTicker.C:
					a.reportMetrics(ctx, currentMonitor.GetMetricsStorage(), gatherChan)
				case <-ctx.Done():
					a.log.Infof("App metrics monitor worker exited. Reason: %v", ctx.Err())
					return
				}
			}
		}()
	}

	retryExecutor := common.NewCommonRetryExecutor(ctx, 2*time.Second, 3, nil)
	wg.Add(int(a.config.RateLimit))
	for i := 0; i < int(a.config.RateLimit); i++ {
		i := i
		go func() {
			defer wg.Done()
			w := newWorker(&a.config, a.log)
			for {
				select {
				case data := <-gatherChan:
					err := retryExecutor.RetryOnError(func() error {
						return w.reportDataJSON(data)
					})
					if err != nil {
						a.log.Errorf("failed to report metrics: %v", err)
					}
				case <-ctx.Done():
					a.log.Infof("Sender worker %v exited. Reason: %v", i, ctx.Err())
					return
				}
			}
		}()
	}

	wg.Wait()
}
