package worker

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/fuzzy-toozy/metrics-service/internal/compression"
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
