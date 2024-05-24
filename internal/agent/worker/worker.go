package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/fuzzy-toozy/metrics-service/internal/agent/config"
	"github.com/fuzzy-toozy/metrics-service/internal/agent/mutator"
	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/compression"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
)

type DataType int

const (
	BULK DataType = iota
	SINGLE
)

type ReportData struct {
	Data  json.Marshaler
	DType DataType
}

type AgentWorker interface {
	ReportData(data ReportData) error
}

func GetCompressedBytes(algo string, data *bytes.Buffer) ([]byte, error) {
	factory, err := compression.GetCompressorFactory(algo)
	if err != nil {
		return nil, fmt.Errorf("failed to get compression factory: %w", err)
	}
	compressionBuf := bytes.NewBuffer(make([]byte, 0, len(data.Bytes())))
	compressor, err := factory(compressionBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to create compressor: %w", err)
	}

	_, err = compressor.Write(data.Bytes())

	if err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	err = compressor.Close()

	if err != nil {
		return nil, fmt.Errorf("failed to finalize compressor: %w", err)
	}

	return compressionBuf.Bytes(), nil
}

func WithCompression(m *mutator.DataMutator, conf *config.Config) {
	m.AddFunc(func(ctx context.Context, data *bytes.Buffer) (context.Context, error) {
		compressedData, err := GetCompressedBytes(conf.CompressAlgo, data)
		if err != nil {
			return ctx, fmt.Errorf("failed to compress request data: %w", err)
		}

		*data = *bytes.NewBuffer(compressedData)

		return ctx, nil
	})
}

func WithSignature(m *mutator.DataMutator, conf *config.Config) {
	m.AddFunc(func(ctx context.Context, data *bytes.Buffer) (context.Context, error) {
		hash, err := encryption.SignData(data.Bytes(), conf.SecretKey)
		if err != nil {
			return ctx, fmt.Errorf("failed to sign request data: %w", err)
		}

		ctx = m.AppendCtx(ctx, common.SighashKey, hash)

		return ctx, nil
	})
}

func WithEncryption(m *mutator.DataMutator, conf *config.Config) {
	m.AddFunc(func(ctx context.Context, data *bytes.Buffer) (context.Context, error) {
		_, err := encryption.EncryptRequestBody(data, conf.EncPublicKey)
		if err != nil {
			return ctx, fmt.Errorf("failed to encrypt request data: %w", err)
		}

		return ctx, nil
	})
}
