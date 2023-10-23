package metrics

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Metric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

const (
	GaugeMetricType   = "gauge"
	CounterMetricType = "counter"
)

var supportedMetricTypes map[string]bool = map[string]bool{
	GaugeMetricType:   true,
	CounterMetricType: true,
}

func IsValidMetricType(mtype string) bool {
	_, ok := supportedMetricTypes[mtype]
	return ok
}

func (m *Metric) GetData() (string, error) {
	mt := strings.ToLower(m.MType)
	var err error
	var res string

	if mt == GaugeMetricType {
		if m.Value != nil {
			res = strconv.FormatFloat(*m.Value, 'f', -1, 64)
		} else {
			err = fmt.Errorf("no value for metric '%v' of type %v", m.ID, mt)
		}
	} else if mt == CounterMetricType {
		if m.Delta != nil {
			res = strconv.FormatInt(*m.Delta, 10)
		} else {
			err = fmt.Errorf("no value for metric '%v' of type %v", m.ID, mt)
		}
	} else {
		err = fmt.Errorf("invalid metric type '%v' for metric '%v'", mt, m.ID)
	}

	return res, err
}

func (m *Metric) UpdateData(data string) error {
	mt := strings.ToLower(m.MType)

	if mt == GaugeMetricType {
		res, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return fmt.Errorf("invalid update data '%v' for metric '%v' of type '%v'", data, m.ID, m.MType)
		}
		m.Value = &res
	} else if mt == CounterMetricType {
		res, err := strconv.ParseInt(data, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid update data '%v' for metric '%v' of type '%v'", data, m.ID, m.MType)
		}
		var prev int64
		if m.Delta != nil {
			prev = *m.Delta
		}
		res += prev
		m.Delta = &res
	} else {
		return fmt.Errorf("invalid metric type '%v' for metric '%v'", mt, m.ID)
	}

	return nil
}

func (m *Metric) SetData(data string) error {
	mt := strings.ToLower(m.MType)

	if mt == GaugeMetricType {
		res, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return fmt.Errorf("invalid data '%v' for metric '%v' of type '%v'", data, m.ID, m.MType)
		}
		m.Value = &res
	} else if mt == CounterMetricType {
		res, err := strconv.ParseInt(data, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid data '%v' for metric '%v' of type '%v'", data, m.ID, m.MType)
		}
		m.Delta = &res
	} else {
		return fmt.Errorf("invalid metric type '%v' for metric '%v'", mt, m.ID)
	}

	return nil
}

func NewCounterMetric(id string, val int64) Metric {
	return Metric{ID: id, MType: CounterMetricType, Delta: &val}
}

func NewGaugeMetric(id string, val float64) Metric {
	return Metric{ID: id, MType: GaugeMetricType, Value: &val}
}

func NewMetric(id string, data string, mtype string) (Metric, error) {
	if !IsValidMetricType(mtype) {
		return Metric{}, errors.New(fmt.Sprintf("invalid metric type '%v' for metric '%v'", mtype, id))
	}
	m := Metric{ID: id, MType: mtype}
	err := m.UpdateData(data)

	if err != nil {
		return Metric{}, err
	}

	return m, nil
}
