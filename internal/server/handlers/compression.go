package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fuzzy-toozy/metrics-service/internal/compression"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
)

type CompressorRespWriter struct {
	http.ResponseWriter
	writer io.WriteCloser
}

func (w *CompressorRespWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

func setupCompression(w http.ResponseWriter, r *http.Request, log logging.Logger) (*CompressorRespWriter, error) {
	accEnc := r.Header.Values("Accept-Encoding")
	var factory compression.CompressorFactory
	var compAlgo string
	var err error
	for _, algo := range compression.GetSupportedAlgorithms() {
		for _, encVal := range accEnc {
			if strings.Contains(encVal, algo) {
				compAlgo = algo
				factory, err = compression.GetCompressorFactory(algo)
				if err != nil {
					log.Debugf("Failed to get compression factory for algo %v: %v", compAlgo, err)
					continue
				}
				goto exitLoop
			}
		}
	}

exitLoop:
	if err != nil {
		return nil, err
	}

	if factory == nil {
		return nil, fmt.Errorf("no compression algorithm requested by client is supporetd by server")
	}

	compressor, err := factory(w)

	if err != nil {
		return nil, err
	}

	w.Header().Set("Content-Encoding", compAlgo)
	return &CompressorRespWriter{ResponseWriter: w, writer: compressor}, nil
}

func needToDecompress(r *http.Request) bool {
	return len(r.Header.Get("Content-Encoding")) > 0
}

func setupDecompression(r *http.Request) error {
	var decompressor io.ReadCloser
	contentEncoding := r.Header.Get("Content-Encoding")
	factory, err := compression.GetDeompressorFactory(contentEncoding)
	if err != nil {
		return fmt.Errorf("failed to get compression factory for encoding %v: %w", contentEncoding, err)
	}
	decompressor, err = factory(r.Body)
	if err != nil {
		return fmt.Errorf("failed to create decompressor for encoding %v: %w", contentEncoding, err)
	}

	r.Body = decompressor

	return nil
}

func WithCompression(h http.Handler, log logging.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respWriter := w
		respCompressorWriter, err := setupCompression(w, r, log)
		if err != nil {
			log.Debugf("Failed to setup compression: %v", err)
		} else {
			respWriter = respCompressorWriter
		}

		if needToDecompress(r) {
			err = setupDecompression(r)
			if err != nil {
				log.Debugf("Failed to setup decompression")
			}
		}

		defer func() {
			if respCompressorWriter != nil && respCompressorWriter.writer != nil {
				err := respCompressorWriter.writer.Close()
				if err != nil {
					log.Errorf("Compressor close error: %v", err)
				}
			}
		}()

		h.ServeHTTP(respWriter, r)
	})
}
