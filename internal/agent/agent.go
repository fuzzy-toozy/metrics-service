// Package agent Metrics gathering agent.
// Gathers various memory and CPU metrics from local machine
// and sends them to speficied server.
package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	monitorHttp "github.com/fuzzy-toozy/metrics-service/internal/agent/http"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/worker"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
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

func (a *Agent) reportMetrics(ctx context.Context, mstorage storage.MetricsStorage, gatherChan chan<- worker.ReportData) {
	allMetrics := mstorage.GetAllMetrics()

	if len(allMetrics) == 0 {
		return
	}

	rData := worker.ReportData{
		Data:  allMetrics,
		DType: worker.BULK,
	}

	select {
	case gatherChan <- rData:
	case <-ctx.Done():
		return
	}

	for _, m := range allMetrics {
		rData := worker.ReportData{
			DType: worker.SINGLE,
			Data:  storage.StorageMetric(m),
		}
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

	gatherChan := make(chan worker.ReportData, a.config.RateLimit)
	defer close(gatherChan)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
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
			reportTicker := time.NewTicker(a.config.ReportInterval.D)
			for {
				select {
				case <-time.After(a.config.PollInterval.D):
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

	const defaultRetryTimeout = 2
	const defaultRetriesCount = 3
	retryExecutor := common.NewCommonRetryExecutor(ctx, defaultRetryTimeout*time.Second, defaultRetriesCount, nil)
	wg.Add(int(a.config.RateLimit))
	for i := 0; i < int(a.config.RateLimit); i++ {
		i := i
		var w worker.AgentWorker
		if a.config.ClientType == common.ModeHTTP {
			w = worker.NewWorkerHTTP(&a.config, a.log, monitorHttp.NewDefaultHTTPClient())
		} else {
			work, err := worker.NewWorkerGRPC(&a.config)
			if err != nil {
				a.log.Fatalf("Failed to create GRPC worker: %w", err)
			}
			w = work
		}

		go func() {
			defer wg.Done()

			for {
				select {
				case data := <-gatherChan:
					err := retryExecutor.RetryOnError(func() error {
						return w.ReportData(data)
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
