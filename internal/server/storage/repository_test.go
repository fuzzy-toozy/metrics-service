package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsStorage(t *testing.T) {
	storage := NewCommonMetricsStorage()
	repoName := "GaugeRepo"

	require.NoError(t, storage.AddRepository(repoName, NewGaugeMetricRepository()))

	repo, err := storage.GetRepository("AbsentRepo")
	require.Error(t, err)
	require.Nil(t, repo)

	repo, err = storage.GetRepository(repoName)
	require.NoError(t, err)
	require.NotNil(t, repo)

	err = storage.DeleteRepository(repoName)
	require.NoError(t, err)

	repo, err = storage.GetRepository(repoName)
	require.Error(t, err)
	require.Nil(t, repo)
}

type MetricTestData struct {
	metricName           string
	metricInitialVal     string
	metricUpdateVal      string
	metricAfterUpdateVal string
	invalidMetricVals    []string
}

func generalRepoTest(t *testing.T, repo Repository, data ...MetricTestData) {

	for _, d := range data {
		for _, v := range d.invalidMetricVals {
			require.Error(t, repo.AddOrUpdate(d.metricName, v))
		}

		require.NoError(t, repo.AddOrUpdate(d.metricName, d.metricInitialVal))

		metric, err := repo.Get(d.metricName)
		require.NoError(t, err)
		require.Equal(t, metric.GetValue(), d.metricInitialVal)

		require.NoError(t, repo.AddOrUpdate(d.metricName, d.metricUpdateVal))

		metric, err = repo.Get(d.metricName)
		require.NoError(t, err)
		require.Equal(t, metric.GetValue(), d.metricAfterUpdateVal)

		require.NoError(t, repo.Delete(d.metricName))

		metric, err = repo.Get(d.metricName)
		require.Error(t, err)
		require.Nil(t, metric)

		require.NoError(t, repo.Delete(d.metricName))
	}
}

func TestMetricsRepo(t *testing.T) {
	data := []MetricTestData{
		{
			metricName:           "m1",
			metricInitialVal:     "100",
			metricUpdateVal:      "200",
			metricAfterUpdateVal: "200",
			invalidMetricVals:    []string{"inv"},
		},
		{
			metricName:           "m2",
			metricInitialVal:     "9999",
			metricUpdateVal:      "10000",
			metricAfterUpdateVal: "10000",
			invalidMetricVals:    []string{"vni"},
		},
	}
	generalRepoTest(t, NewGaugeMetricRepository(), data...)

	data = []MetricTestData{
		{
			metricName:           "m1",
			metricInitialVal:     "100",
			metricUpdateVal:      "200",
			metricAfterUpdateVal: "300",
			invalidMetricVals:    []string{"inv, 20.20"},
		},
		{
			metricName:           "m2",
			metricInitialVal:     "9999",
			metricUpdateVal:      "10000",
			metricAfterUpdateVal: "19999",
			invalidMetricVals:    []string{"vni, 9999.9999"},
		},
	}
	generalRepoTest(t, NewCounterMetricRepository(), data...)
}
