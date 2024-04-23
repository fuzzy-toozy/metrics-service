package exitcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestMainOsExit(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "./...")
}
