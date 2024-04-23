package errcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrCheck(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "./...")
}
