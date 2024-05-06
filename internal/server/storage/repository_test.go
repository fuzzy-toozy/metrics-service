package storage

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/beevik/guid"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MetricTestData struct {
	name              string
	mtype             string
	initialVal        string
	updateVal         string
	afterUpdateVal    string
	invalidMetricVals []string
}

func Test_MetricsRepo(t *testing.T) {
	repo := NewCommonMetricsRepository()
	data := []MetricTestData{
		{
			name:              "m1",
			mtype:             metrics.GaugeMetricType,
			initialVal:        "100",
			updateVal:         "200",
			afterUpdateVal:    "200",
			invalidMetricVals: []string{"inv"},
		},
		{
			name:              "m2",
			mtype:             metrics.GaugeMetricType,
			initialVal:        "9999",
			updateVal:         "10000",
			afterUpdateVal:    "10000",
			invalidMetricVals: []string{"vni"},
		},
		{
			name:              "m1",
			mtype:             metrics.CounterMetricType,
			initialVal:        "100",
			updateVal:         "200",
			afterUpdateVal:    "300",
			invalidMetricVals: []string{"inv, 20.20"},
		},
		{
			name:              "m2",
			mtype:             metrics.CounterMetricType,
			initialVal:        "9999",
			updateVal:         "10000",
			afterUpdateVal:    "19999",
			invalidMetricVals: []string{"vni, 9999.9999"},
		},
	}

	_, err := repo.AddOrUpdate("k", "v", "garbage")
	assert.Error(t, err)

	_, err = repo.AddOrUpdate("k", "v", metrics.CounterMetricType)
	assert.Error(t, err)

	_, err = repo.Get("k", "garbage")
	assert.Error(t, err)

	num := int(rand.Int31()) % 100
	check := make(map[string]int64, num)
	for i := 0; i < num; i++ {
		id := guid.NewString()
		valInt := rand.Int63()
		check[id] = valInt

		valStr := strconv.FormatInt(valInt, 10)
		var v string
		v, err = repo.AddOrUpdate(id, valStr, metrics.CounterMetricType)
		require.NoError(t, err)
		require.Equal(t, v, valStr)
	}

	repoMetrics, err := repo.GetAll()
	require.NoError(t, err)

	for _, m := range repoMetrics {
		v, ok := check[m.ID]
		require.True(t, ok)
		require.NotNil(t, m.Delta)
		assert.Equal(t, v, *m.Delta)
	}

	for _, d := range data {
		for _, v := range d.invalidMetricVals {
			_, err := repo.AddOrUpdate(d.name, v, d.mtype)
			require.Error(t, err)
		}

		updatedVal, err := repo.AddOrUpdate(d.name, d.initialVal, d.mtype)
		require.NoError(t, err)

		metric, err := repo.Get(d.name, d.mtype)
		require.NoError(t, err)
		data, err := metric.GetData()
		require.NoError(t, err)
		require.Equal(t, data, updatedVal)
		require.Equal(t, data, d.initialVal)

		updatedVal, err = repo.AddOrUpdate(d.name, d.updateVal, d.mtype)
		require.NoError(t, err)

		metric, err = repo.Get(d.name, d.mtype)
		require.NoError(t, err)
		data, err = metric.GetData()
		require.NoError(t, err)
		require.Equal(t, data, updatedVal)
		require.Equal(t, data, d.afterUpdateVal)

		require.NoError(t, repo.Delete(d.name))

		metric, err = repo.Get(d.name, d.mtype)
		require.Error(t, err)

		require.NoError(t, repo.Delete(d.name))
	}
}
