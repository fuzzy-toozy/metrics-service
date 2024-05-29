package worker

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	monitorHttp "github.com/fuzzy-toozy/metrics-service/internal/agent/http"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type WorkerHTTP struct {
	log           log.Logger
	httpClient    monitorHttp.HTTPClient
	config        *config.Config
	reportDataBuf *bytes.Buffer
}

var _ AgentWorker = (*WorkerHTTP)(nil)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

var _ http.RoundTripper = (RoundTripperFunc)(nil)

func WithCompression(baseTransport http.RoundTripper, algo string) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			data, err := io.ReadAll(req.Body)

			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}

			compressedData, err := GetCompressedBytes(algo, bytes.NewBuffer(data))
			if err != nil {
				return nil, fmt.Errorf("failed to compress request data: %w", err)
			}

			req.Header.Set("Content-Encoding", algo)

			req.Body = io.NopCloser(bytes.NewReader(compressedData))
			req.ContentLength = int64(len(compressedData))
		}

		return baseTransport.RoundTrip(req)
	})
}

func WithSignature(baseTransport http.RoundTripper, key []byte) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			data, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}

			hash, err := encryption.SignData(data, key)
			if err != nil {
				return nil, fmt.Errorf("failed to sign request data: %w", err)
			}

			req.Header.Set(common.SighashKey, hash)
			req.Body = io.NopCloser(bytes.NewReader(data))
		}

		return baseTransport.RoundTrip(req)
	})
}

func WithEncryption(baseTransport http.RoundTripper, key *rsa.PublicKey) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			data, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}

			encData, err := encryption.EncryptRequestBody(bytes.NewBuffer(data), key)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt request data: %w", err)
			}

			req.Body = io.NopCloser(encData)
			req.ContentLength = int64(len(data))

		}
		return baseTransport.RoundTrip(req)
	})
}

func WithHostIP(baseTransport http.RoundTripper, hostIP string) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.Header.Set(common.IPAddrKey, hostIP)
		return baseTransport.RoundTrip(req)
	})
}

func NewWorkerHTTP(config *config.Config, logger log.Logger, client monitorHttp.HTTPClient) *WorkerHTTP {
	transport := http.DefaultTransport

	if len(config.CompressAlgo) > 0 {
		transport = WithCompression(transport, config.CompressAlgo)
	}

	if config.EncPublicKey != nil {
		transport = WithEncryption(transport, config.EncPublicKey)
	}

	if len(config.SecretKey) > 0 {
		transport = WithSignature(transport, config.SecretKey)
	}

	if len(config.HostIPAddr) > 0 {
		transport = WithHostIP(transport, config.HostIPAddr)
	}

	client.SetTransport(transport)

	w := WorkerHTTP{
		httpClient:    client,
		log:           logger,
		config:        config,
		reportDataBuf: bytes.NewBuffer(nil),
	}

	return &w
}

func (w *WorkerHTTP) ReportData(data ReportData) error {
	var reportURL string
	if data.DType == BULK {
		reportURL = w.config.ReportBulkEndpoint
	} else if data.DType == SINGLE {
		reportURL = w.config.ReportEndpoint
	} else {
		return fmt.Errorf("wrong report data type %v", data.DType)
	}

	defer w.reportDataBuf.Reset()

	if err := json.NewEncoder(w.reportDataBuf).Encode(data.Data); err != nil {
		return fmt.Errorf("failed to encode metrics to JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, reportURL, w.reportDataBuf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Send(req)

	if resp != nil {
		defer func() {
			_, err = io.Copy(io.Discard, resp.Body)
			if err != nil {
				w.log.Debugf("Failed reading request body: %v", err)
			}
			err = resp.Body.Close()
			if err != nil {
				w.log.Debugf("Failed to close resp body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("failed to send metrics. Status code: %v", resp.StatusCode)
		}
	}

	return err
}
