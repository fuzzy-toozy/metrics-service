package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	monitorHttp "github.com/fuzzy-toozy/metrics-service/internal/agent/http"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/mutator"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
)

type WorkerHTTP struct {
	log           log.Logger
	httpClient    monitorHttp.HTTPClient
	config        *config.Config
	dataMutator   *mutator.DataMutator
	reportDataBuf *bytes.Buffer
}

var _ AgentWorker = (*WorkerHTTP)(nil)

func NewWorkerHTTP(config *config.Config, logger log.Logger, client monitorHttp.HTTPClient) *WorkerHTTP {

	dataMutator := mutator.NewDataMutator(func(ctx context.Context, key mutator.ContextKey, val string) context.Context {
		return context.WithValue(ctx, key, val)
	})

	if len(config.SecretKey) > 0 {
		WithSignature(dataMutator, config)
	}

	if config.EncPublicKey != nil {
		WithEncryption(dataMutator, config)
	}

	if len(config.CompressAlgo) > 0 {
		WithCompression(dataMutator, config)
	}

	w := WorkerHTTP{
		httpClient:    client,
		log:           logger,
		config:        config,
		dataMutator:   dataMutator,
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

	ctx, err := w.dataMutator.Run(context.Background(), w.reportDataBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to run request data mutation chain: %w", err)
	}

	dataBuf := w.dataMutator.GetData()
	req, err := http.NewRequest(http.MethodPost, reportURL, dataBuf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", w.config.CompressAlgo)
	req.Header.Set("X-Real-IP", w.config.HostIPAddr)

	sigHash := ""
	sigHashData := ctx.Value(common.SighashKey)
	if sigHashData != nil {
		sigHash = sigHashData.(string)
	}

	if len(sigHash) != 0 {
		req.Header.Set("HashSHA256", sigHash)
	}

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
