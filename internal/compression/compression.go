// Package compression Compression/Decompression support to use in handlers
package compression

import (
	"compress/gzip"
	"fmt"
	"io"
)

const CompressAlgoKey = "CompressAlgo"

type CompressorFactory func(w io.Writer) (io.WriteCloser, error)
type DecompressorFactory func(w io.Reader) (io.ReadCloser, error)

var compressors map[string]CompressorFactory = make(map[string]CompressorFactory)
var decompressors map[string]DecompressorFactory = make(map[string]DecompressorFactory)

func init() {
	compressors["gzip"] = newGzipCompressor
	decompressors["gzip"] = newGzipDecompressor
}

func newGzipCompressor(w io.Writer) (io.WriteCloser, error) {
	return gzip.NewWriterLevel(w, gzip.BestSpeed)
}

func newGzipDecompressor(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

func GetCompressorFactory(name string) (CompressorFactory, error) {
	factory, ok := compressors[name]
	if !ok {
		return nil, fmt.Errorf("compression algorithm %v is not supported", name)
	}

	return factory, nil
}

func GetDeompressorFactory(name string) (DecompressorFactory, error) {
	factory, ok := decompressors[name]
	if !ok {
		return nil, fmt.Errorf("decompression algorithm %v is not supported", name)
	}

	return factory, nil
}

func GetSupportedAlgorithms() []string {
	supportedAlgos := make([]string, len(compressors))
	idx := 0
	for k := range compressors {
		supportedAlgos[idx] = k
		idx++
	}
	return supportedAlgos
}

func GetSupportedContentEncodings() []string {
	ct := make([]string, 1)
	ct[0] = "gzip"
	return ct
}
