// General metric type. Used in agent and server.
// Currently supports 2 metric types: Gauge and Counter.
// Gauge metric type supports only float64 values,
// updates of gauge metric simply replace old value.
// Counter metric type support int64 values,
// updates of counter metric add passed value to previous value.
package metrics

import (
	"fmt"
	"strconv"
	"strings"
)

type Metric struct {
	// ID metric name
	ID string `json:"id"`
	// MType metric type (Gauge or Counter)
	MType string `json:"type"`
	// Delta metric value used for Counter metric type
	Delta *int64 `json:"delta,omitempty"`
	// Value metric value used for Gauge metric type
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

// IsValidMetricType check if metric type is supported.
func IsValidMetricType(mtype string) bool {
	_, ok := supportedMetricTypes[mtype]
	return ok
}

// Equal compares two metrics.
// Metrics are considered equal if types are equal
// and according values for types are equal.
// Metrics with invalid types considered unequal.
// Metrics with unset values considered equal.
func (m *Metric) Equal(arg *Metric) bool {
	if arg.MType != m.MType {
		return false
	}

	switch m.MType {
	case CounterMetricType:
		if m.Delta == nil && arg.Delta == nil {
			return true
		}

		if m.Delta != nil && arg.Delta != nil {
			return *m.Delta == *arg.Delta
		}

		return false
	case GaugeMetricType:
		if m.Value == nil && arg.Value == nil {
			return true
		}

		if m.Value != nil && arg.Value != nil {
			return *m.Value == *m.Value
		}

		return false
	}

	return false
}

// GetData get string representation of metric value.
// Returns error if metric is of invalid type or value is not set.
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

// UpdateData updates meric value from string.
// Returns error if string value parse to according type failed or
// metric type is invalid.
// For gauge type simply replaces the value,
// for counter type adds new value to previos value.
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

// SetData sets metric value from string.
// Returns error if string value parse to according type failed or
// metric type is invalid.
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
		return Metric{}, fmt.Errorf("invalid metric type '%v' for metric '%v'", mtype, id)
	}
	m := Metric{ID: id, MType: mtype}
	err := m.UpdateData(data)

	if err != nil {
		return Metric{}, err
	}

	return m, nil
}
