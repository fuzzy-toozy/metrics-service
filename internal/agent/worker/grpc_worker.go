package worker

import (
	"bytes"
	"context"
	"fmt"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/monitor/storage"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/mutator"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/grpcconv"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type WorkerGRPC struct {
	config        *config.Config
	dataMutator   *mutator.DataMutator
	reportDataBuf *bytes.Buffer
	client        pb.MetricsServiceClient
}

var _ AgentWorker = (*WorkerGRPC)(nil)

func NewWorkerGRPC(config *config.Config) (*WorkerGRPC, error) {
	opts := []grpc.DialOption{}

	if config.CompressAlgo == "gzidp" {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	}
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(config.ServerAddress, opts...)
	if err != nil {
		return nil, err
	}

	dataMutator := mutator.NewDataMutator(func(ctx context.Context, key mutator.ContextKey, val string) context.Context {
		return metadata.AppendToOutgoingContext(ctx, string(key), string(val))
	})

	if len(config.SecretKey) > 0 {
		WithSignature(dataMutator, config)
	}

	if config.EncPublicKey != nil {
		WithEncryption(dataMutator, config)
	}

	w := WorkerGRPC{
		client:        pb.NewMetricsServiceClient(conn),
		config:        config,
		dataMutator:   dataMutator,
		reportDataBuf: bytes.NewBuffer(nil),
	}

	return &w, nil
}

func (w *WorkerGRPC) ReportData(data ReportData) error {
	var reportDataBuf *bytes.Buffer
	if data.DType == BULK {
		reportMetrics := data.Data.(storage.StorageMetrics)
		metrics := &pb.Metrics{}
		for _, m := range reportMetrics {
			metrics.Metrics = append(metrics.Metrics, grpcconv.MetricToGRPC(m))
		}

		data, err := proto.Marshal(metrics)
		if err != nil {
			return fmt.Errorf("failed to marshal grpc request: %w", err)
		}

		reportDataBuf = bytes.NewBuffer(data)
	} else if data.DType == SINGLE {
		reportMetric := data.Data.(storage.StorageMetric)
		data, err := proto.Marshal(grpcconv.MetricToGRPC(metrics.Metric(reportMetric)))
		if err != nil {
			return fmt.Errorf("failed to marshal grpc request: %w", err)
		}
		reportDataBuf = bytes.NewBuffer(data)
	} else {
		return fmt.Errorf("wrong report data type %v", data.DType)
	}

	ctx, err := w.dataMutator.Run(context.Background(), reportDataBuf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to run request data mutation chain: %w", err)
	}

	if len(w.config.HostIPAddr) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, common.IPAddrKey, w.config.HostIPAddr)
	}

	dataBuf := w.dataMutator.GetData()

	if data.DType == BULK {
		_, err = w.client.UpdateMetrics(ctx, &pb.UpdateRequest{
			Data: dataBuf.Bytes(),
		})
	} else if data.DType == SINGLE {
		_, err = w.client.UpdateMetric(ctx, &pb.UpdateRequest{
			Data: dataBuf.Bytes(),
		})
	}

	if err != nil {
		return fmt.Errorf("failed to perform GRPC request: %w", err)
	}

	return nil
}
