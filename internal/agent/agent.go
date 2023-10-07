package agent

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/monitor"
)

type HTTPClient interface {
	Send(r *http.Request) (*http.Response, error)
}

type DefaultHTTPClient struct {
	client http.Client
}

type DurationOption struct {
	d time.Duration
}

func (o *DurationOption) String() string {
	return o.d.String()
}

func (o *DurationOption) Set(flagValue string) error {
	intD, err := strconv.ParseInt(flagValue, 10, 64)
	if err != nil {
		return err
	}
	o.d = time.Duration(intD) * time.Second
	return nil
}

type Config struct {
	serverAddress  string
	reportURL      string
	pollInterval   DurationOption
	reportInterval DurationOption
}

func ParseCmdFlags() *Config {
	c := Config{}
	c.pollInterval.d = 2 * time.Second
	c.reportInterval.d = 10 * time.Second
	flag.StringVar(&c.serverAddress, "a", "localhost:8080", "Server address")
	flag.StringVar(&c.reportURL, "u", "/update", "Server endpoint path")
	flag.Var(&c.pollInterval, "p", "Metrics polling interval(seconds)")
	flag.Var(&c.reportInterval, "r", "Metrics report interval(seconds)")
	flag.Parse()

	return &c
}

func (c *DefaultHTTPClient) Send(r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}

func NewDefaultHTTPClient() *DefaultHTTPClient {
	c := DefaultHTTPClient{client: http.Client{
		Timeout: 30 * time.Second,
	}}
	return &c
}

type Agent struct {
	metricsMonitor monitor.Monitor
	httpClient     HTTPClient
	log            log.Logger
	config         Config
}

func NewAgent(config Config, httpClient HTTPClient, metricsMonitor monitor.Monitor, logger log.Logger) *Agent {
	a := Agent{config: config, httpClient: httpClient, metricsMonitor: metricsMonitor, log: logger}
	return &a
}

func (a *Agent) ReportMetrics() error {
	serverEndpoint := a.config.serverAddress + a.config.reportURL
	serverEndpoint = path.Clean(serverEndpoint)
	serverEndpoint = strings.Trim(serverEndpoint, "/")
	serverEndpoint = fmt.Sprintf("http://%v", serverEndpoint)
	return a.metricsMonitor.GetMetrics().ForEachMetric(func(metricName string, m monitor.Metric) error {
		metricValue := m.GetValue()
		metricType := m.GetType()
		url := fmt.Sprintf("%v/%v/%v/%v", serverEndpoint, metricType, metricName, metricValue)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return err
		}
		resp, err := a.httpClient.Send(req)

		if resp != nil {
			defer func() {
				_, err := io.Copy(io.Discard, resp.Body)
				if err != nil {
					a.log.Debugf("Failed reading request body: %v", err)
				}
				resp.Body.Close()
			}()

			a.log.Debugf("Sent metric of type %v, name %v, value %v to %v. Status %v",
				metricType, metricName, metricValue, url, resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("failed to send metric %v. Status code: %v", metricName, resp.StatusCode)
			}
		}

		return err
	})
}

func (a *Agent) Run() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-time.After(2 * time.Second):
			err := a.metricsMonitor.GatherMetrics()
			if err != nil {
				a.log.Warnf("Failed to gather metrics. %v", err)
			}
		case <-ticker.C:
			err := a.ReportMetrics()
			if err != nil {
				a.log.Warnf("Failed to report metric. %v", err)
			}
		}
	}
}
