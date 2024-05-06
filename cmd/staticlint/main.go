package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fuzzy-toozy/metrics-service/internal/checkers/errcheck"
	"github.com/fuzzy-toozy/metrics-service/internal/checkers/exitcheck"
	"github.com/fuzzy-toozy/metrics-service/internal/codegen/passes"
	"github.com/julz/importas"
	magicNumbers "github.com/tommy-muehle/go-mnd/v2"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

type lintConf struct {
	StaticCheck []string `json:"staticheck"`
	StyleCheck  []string `json:"stylecheck"`
	Simple      []string `json:"simple"`
	Custom      []string `json:"custom"`
}

func main() {
	var mychecks []*analysis.Analyzer

	c := lintConf{}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("Could not get filename")
		return
	}

	dir := filepath.Dir(filename)

	confFile, err := os.Open(filepath.Join(dir, "lint.conf"))
	if err != nil {
		fmt.Printf("Failed to open config file: %v\n", err)
		return
	}

	err = json.NewDecoder(confFile).Decode(&c)
	if err != nil {
		fmt.Printf("Failed to decode config file: %v\n", err)
		return
	}

	enabledStatic := make(map[string]bool, len(c.StaticCheck))
	for _, name := range c.StaticCheck {
		enabledStatic[name] = true
	}

	enabledSimple := make(map[string]bool, len(c.Simple))
	for _, name := range c.Simple {
		enabledSimple[name] = true
	}

	enabledStyle := make(map[string]bool, len(c.StyleCheck))
	for _, name := range c.StyleCheck {
		enabledStyle[name] = true
	}

	for _, v := range staticcheck.Analyzers {
		if enabledStatic[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	for _, v := range simple.Analyzers {
		if enabledSimple[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	for _, v := range stylecheck.Analyzers {
		if enabledStyle[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	getCustomAnalyzersMap := func(analyzers ...*analysis.Analyzer) map[string]*analysis.Analyzer {
		m := make(map[string]*analysis.Analyzer, len(analyzers))
		for _, analyzer := range analyzers {
			m[analyzer.Name] = analyzer
		}

		return m
	}

	customAnalyzersMap := getCustomAnalyzersMap(errcheck.Analyzer,
		exitcheck.Analyzer,
		// A vet analyzer to detect magic numbers.
		magicNumbers.Analyzer,
		// A linter to enforce importing certain packages consistently.
		importas.Analyzer)

	for _, name := range c.Custom {
		if a, ok := customAnalyzersMap[name]; ok {
			mychecks = append(mychecks, a)
		}
	}

	passes := passes.GetAnalysers()

	mychecks = append(mychecks, passes...)

	multichecker.Main(
		mychecks...,
	)
}
