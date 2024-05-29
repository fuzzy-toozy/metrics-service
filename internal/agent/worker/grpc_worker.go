package worker

import (
	"bytes"
	"context"
	"fmt"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/grpcconv"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

type WorkerGRPC struct {
	config        *config.Config
	reportDataBuf *bytes.Buffer
	client        pb.MetricsServiceClient
}

var _ AgentWorker = (*WorkerGRPC)(nil)

func NewWorkerGRPC(config *config.Config) (*WorkerGRPC, error) {
	opts := []grpc.DialOption{}

	if config.CompressAlgo == "gzip" {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	}

	if len(config.CACertPath) > 0 {
		creds, err := encryption.SetupClientTLS(config.CACertPath, config.EncKeyPath, config.AgentCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to setup TLS for grpc client: %w", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(config.ServerAddress, opts...)
	if err != nil {
		return nil, err
	}

	w := WorkerGRPC{
		client:        pb.NewMetricsServiceClient(conn),
		config:        config,
		reportDataBuf: bytes.NewBuffer(nil),
	}

	return &w, nil
}

func (w *WorkerGRPC) ReportData(data ReportData) error {
	var err error
	ctx := context.Background()
	if len(w.config.HostIPAddr) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, common.IPAddrKey, w.config.HostIPAddr)
	}

	if data.DType == BULK {
		reportMetrics := data.Data.(storage.StorageMetrics)
		sendM := grpcconv.MetricsToGRPC(reportMetrics)
		_, err = w.client.UpdateMetrics(ctx, &pb.MetricsUpdateRequest{
			Metrics: sendM,
		})
	} else if data.DType == SINGLE {
		reportMetric := data.Data.(storage.StorageMetric)
		sendM := grpcconv.MetricToGRPC(metrics.Metric(reportMetric))
		_, err = w.client.UpdateMetric(ctx, &pb.MetricUpdateRequest{
			Metric: sendM,
		})
	} else {
		return fmt.Errorf("wrong report data type %v", data.DType)
	}

	if err != nil {
		return fmt.Errorf("failed to perform GRPC request: %w", err)
	}

	return nil
}
