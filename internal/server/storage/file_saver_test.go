package storage

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/beevik/guid"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getMetricsFromFile(fileName string) ([]metrics.Metric, error) {
	m := make([]metrics.Metric, 0)
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func Test_FileSaver(t *testing.T) {
	repo := NewCommonMetricsRepository()
	outFile := "repoFile.out"
	defer os.Remove(outFile)
	logger := log.NewDevZapLogger()
	fs := NewFileSaver(repo, outFile, logger)

	id := guid.NewString()
	intVal := rand.Int63()
	intStr := strconv.FormatInt(intVal, 10)
	v, err := repo.AddOrUpdate(id, intStr, metrics.CounterMetricType)

	require.NoError(t, err)
	assert.Equal(t, v, intStr)

	err = fs.Save()
	require.NoError(t, err)

	m, err := getMetricsFromFile(outFile)
	require.NoError(t, err)

	decodedM := m[0]
	assert.Equal(t, decodedM.ID, id)
	assert.Equal(t, decodedM.MType, metrics.CounterMetricType)
	require.NotNil(t, decodedM.Delta)
	assert.Equal(t, *decodedM.Delta, intVal)
}

func Test_PeriodicFileSaver(t *testing.T) {
	repo := NewCommonMetricsRepository()
	outFile := "repoFile.out"
	defer os.RemoveAll(outFile)
	logger := log.NewDevZapLogger()

	dur := time.Duration(1+rand.Int31()%500) * time.Millisecond
	fs := NewFileSaver(repo, outFile, logger)
	pfs := NewPeriodicSaver(dur, logger, fs)

	id := guid.NewString()
	intVal := rand.Int63()
	intStr := strconv.FormatInt(intVal, 10)
	v, err := repo.AddOrUpdate(id, intStr, metrics.CounterMetricType)

	require.NoError(t, err)
	assert.Equal(t, v, intStr)

	pfs.Run()

	<-time.After(dur + dur/2)
	pfs.ticker.Stop()

	m, err := getMetricsFromFile(outFile)
	require.NoError(t, err)
	require.Equal(t, len(m), 1)

	decodedM := m[0]
	assert.Equal(t, decodedM.ID, id)
	assert.Equal(t, decodedM.MType, metrics.CounterMetricType)
	require.NotNil(t, decodedM.Delta)
	assert.Equal(t, *decodedM.Delta, intVal)

	assert.NoError(t, repo.Delete(id))

	id = guid.NewString()
	intVal = rand.Int63()
	intStr = strconv.FormatInt(intVal, 10)
	v, err = repo.AddOrUpdate(id, intStr, metrics.CounterMetricType)
	require.NoError(t, err)
	assert.Equal(t, v, intStr)

	pfs.Stop()

	m, err = getMetricsFromFile(outFile)
	require.NoError(t, err)
	require.Equal(t, len(m), 1)

	decodedM = m[0]
	assert.Equal(t, decodedM.ID, id)
	assert.Equal(t, decodedM.MType, metrics.CounterMetricType)
	require.NotNil(t, decodedM.Delta)
	assert.Equal(t, *decodedM.Delta, intVal)
}
