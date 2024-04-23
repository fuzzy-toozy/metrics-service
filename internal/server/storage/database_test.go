package storage

import (
	"bytes"
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/beevik/guid"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/stretchr/testify/require"
)

func Test_Database(t *testing.T) {
	cfg := config.Config{}
	err := cfg.ParseEnvVariables()
	require.NoError(t, err)

	cfg.DatabaseConfig.DriverName = "pgx"
	cfg.DatabaseConfig.PingTimeout = 2 * time.Second

	if len(cfg.DatabaseConfig.ConnString) == 0 {
		return
	}

	stopCtx, stopFunc := context.WithCancel(context.TODO())
	defer stopFunc()
	repo, err := NewPGMetricRepository(cfg.DatabaseConfig, NewDefaultDBRetryExecutor(stopCtx), log.NewDevZapLogger())
	require.NoError(t, err)

	require.NoError(t, repo.HealthCheck())

	m := metrics.NewCounterMetric(guid.NewString(), rand.Int63())
	data, err := m.GetData()
	require.NoError(t, err)

	val, err := repo.AddOrUpdate(m.ID, data, m.MType)
	require.NoError(t, err)
	require.Equal(t, val, data)

	mDB, err := repo.Get(m.ID, m.MType)
	require.NoError(t, err)
	require.True(t, mDB.Equal(&m))

	err = repo.Delete(m.ID)
	require.NoError(t, err)

	metricsNum := int(rand.Int31()) % 1000
	metricsMap := make(map[string]metrics.Metric, metricsNum)
	metricsArr := make([]metrics.Metric, 0, metricsNum)

	for i := 0; i < int(rand.Int31())%1000; i++ {
		id := guid.NewString()
		if i%2 == 0 {
			m = metrics.NewCounterMetric(id, rand.Int63())
		} else {
			m = metrics.NewGaugeMetric(id, rand.Float64())
		}
		metricsMap[id] = m
		metricsArr = append(metricsArr, m)
	}

	err = repo.AddMetricsBulk(metricsArr)
	require.NoError(t, err)

	metricsDB, err := repo.GetAll()
	require.NoError(t, err)
	cnt := 0
	for _, mDB := range metricsDB {
		mTest, ok := metricsMap[mDB.ID]
		if !ok {
			continue
		}
		cnt++
		require.True(t, mTest.Equal(&mDB))
	}

	require.Equal(t, cnt, len(metricsMap))

	require.Error(t, repo.Save(bytes.NewBuffer(make([]byte, 0))))
	require.Error(t, repo.Load(bytes.NewBuffer(make([]byte, 0))))
	_, err = repo.MarshalJSON()
	require.Error(t, err)
	err = repo.UnmarshalJSON(make([]byte, 0))
	require.Error(t, err)

	require.NoError(t, repo.DeleteAll())
	require.NoError(t, repo.Release())
	require.NoError(t, repo.Close())
}
