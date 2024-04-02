package metrics

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/beevik/guid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Number interface {
	int64 | float64
}

type NumberPtr interface {
	*int64 | *float64
}

func Test_ValidMetricType(t *testing.T) {
	require.True(t, IsValidMetricType(GaugeMetricType))
	require.True(t, IsValidMetricType(CounterMetricType))
	require.False(t, IsValidMetricType("garbage"))
}

func Test_CounterMetricConstsructor(t *testing.T) {
	valInt := rand.Int63()
	id := guid.New().String()
	m := NewCounterMetric(id, valInt)
	require.Equal(t, m.ID, id)
	require.Equal(t, m.MType, CounterMetricType)
	assert.NotNil(t, m.Delta)
	require.Equal(t, *m.Delta, valInt)

	valInt = rand.Int63()
	valString := strconv.FormatInt(valInt, 10)
	id = guid.New().String()
	m, err := NewMetric(id, valString, CounterMetricType)
	require.NoError(t, err)
	require.Equal(t, m.ID, id)
	assert.NotNil(t, m.Delta)
	require.Equal(t, *m.Delta, valInt)
	require.Equal(t, m.MType, CounterMetricType)

	m, err = NewMetric(id, valString, "garbage")
	require.Error(t, err)

	m, err = NewMetric(id, "garbage", CounterMetricType)
	require.Error(t, err)
}

func Test_CounterMetricGetSet(t *testing.T) {
	valInt := rand.Int63()
	id := guid.New().String()
	m := NewCounterMetric(id, valInt)

	data, err := m.GetData()
	assert.NoError(t, err)
	require.Equal(t, data, strconv.FormatInt(valInt, 10))

	valInt = rand.Int63()
	valStr := strconv.FormatInt(valInt, 10)
	assert.NoError(t, m.SetData(valStr))
	require.Equal(t, *m.Delta, valInt)

	data, err = m.GetData()
	assert.NoError(t, err)
	require.Equal(t, data, valStr)

	require.Error(t, m.SetData("10.11"))

	require.Error(t, m.SetData("garbage"))

	m.Delta = nil
	_, err = m.GetData()
	require.Error(t, err)

	m.MType = "garbage"
	require.Error(t, m.SetData(valStr))

	_, err = m.GetData()
	require.Error(t, err)
}

func Test_CounterMetricUpdate(t *testing.T) {
	valInt := rand.Int63() % 1000
	id := guid.New().String()
	m := NewCounterMetric(id, valInt)

	data, err := m.GetData()
	assert.NoError(t, err)
	require.Equal(t, data, strconv.FormatInt(valInt, 10))
	valNewInt := rand.Int63() % 1000
	valStr := strconv.FormatInt(valNewInt, 10)

	assert.NoError(t, m.UpdateData(valStr))
	assert.NotNil(t, m.Delta)
	require.Equal(t, *m.Delta, valInt+valNewInt)

	require.Error(t, m.UpdateData("garbage"))

	m.MType = "garbage"
	require.Error(t, m.UpdateData(valStr))
}

func Test_GaugeMetricConstructor(t *testing.T) {
	valFloat := rand.Float64()
	id := guid.New().String()
	m := NewGaugeMetric(id, valFloat)
	require.Equal(t, m.ID, id)
	require.Equal(t, m.MType, GaugeMetricType)
	assert.NotNil(t, m.Value)
	require.Equal(t, *m.Value, valFloat)

	valFloat = rand.Float64()
	valString := strconv.FormatFloat(valFloat, 'f', -1, 64)
	id = guid.New().String()
	m, err := NewMetric(id, valString, GaugeMetricType)
	require.NoError(t, err)
	require.Equal(t, m.ID, id)
	assert.NotNil(t, m.Value)
	require.Equal(t, *m.Value, valFloat)
	require.Equal(t, m.MType, GaugeMetricType)

	m, err = NewMetric(id, valString, "garbage")
	require.Error(t, err)

	m, err = NewMetric(id, "garbage", GaugeMetricType)
	require.Error(t, err)
}

func Test_GaugeMetricGetSet(t *testing.T) {
	valFloat := rand.Float64()
	id := guid.New().String()
	m := NewGaugeMetric(id, valFloat)

	data, err := m.GetData()
	assert.NoError(t, err)
	require.Equal(t, data, strconv.FormatFloat(valFloat, 'f', -1, 64))

	valFloat = rand.Float64()
	valStr := strconv.FormatFloat(valFloat, 'f', -1, 64)
	assert.NoError(t, m.SetData(valStr))
	require.Equal(t, *m.Value, valFloat)

	data, err = m.GetData()
	assert.NoError(t, err)
	require.Equal(t, data, valStr)

	require.Error(t, m.SetData("garbage"))

	m.Value = nil
	_, err = m.GetData()
	require.Error(t, err)

	m.MType = "garbage"
	require.Error(t, m.SetData(valStr))

	_, err = m.GetData()
	require.Error(t, err)

	require.Error(t, m.SetData("garbage"))
}

func Test_GaugeMetricUpdate(t *testing.T) {
	valFloat := rand.Float64()
	id := guid.New().String()
	m := NewGaugeMetric(id, valFloat)

	data, err := m.GetData()
	assert.NoError(t, err)
	require.Equal(t, data, strconv.FormatFloat(valFloat, 'f', -1, 64))
	valNewFloat := rand.Float64()
	valStr := strconv.FormatFloat(valNewFloat, 'f', -1, 64)

	assert.NoError(t, m.UpdateData(valStr))
	assert.NotNil(t, m.Value)
	require.Equal(t, *m.Value, valNewFloat)

	require.Error(t, m.UpdateData("garbage"))

	m.MType = "garbage"
	require.Error(t, m.UpdateData(valStr))
}

func Test_GaugeNewMetric(t *testing.T) {
	valFloat := rand.Float64()
	valString := strconv.FormatFloat(valFloat, 'f', -1, 64)
	id := guid.New().String()
	m, err := NewMetric(id, valString, GaugeMetricType)
	require.NoError(t, err)
	require.Equal(t, m.ID, id)
	assert.NotNil(t, m.Value)
	require.Equal(t, *m.Value, valFloat)
	require.Equal(t, m.MType, GaugeMetricType)
}
