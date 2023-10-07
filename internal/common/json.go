package common

import (
	"fmt"
	"strconv"
	"strings"
)

type MetricJSON struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func (m *MetricJSON) GetData() (string, error) {
	mt := strings.ToLower(m.MType)
	var err error
	var res string

	if mt == MetricTypeGauge {
		if m.Value != nil {
			res = strconv.FormatFloat(*m.Value, 'f', -1, 64)
		} else {
			err = fmt.Errorf("no value for metric of type %v", mt)
		}
	} else if mt == MetricTypeCounter {
		if m.Delta != nil {
			res = strconv.FormatInt(*m.Delta, 10)
		} else {
			err = fmt.Errorf("no value for metric of type %v", mt)
		}
	} else {
		err = fmt.Errorf("wrong metric type %v", mt)
	}

	return res, err
}

func (m *MetricJSON) SetData(data string) error {
	mt := strings.ToLower(m.MType)
	var err error = nil

	if mt == MetricTypeGauge {
		var res float64
		res, err = strconv.ParseFloat(data, 64)
		m.Value = &res
	} else if mt == MetricTypeCounter {
		var res int64
		res, err = strconv.ParseInt(data, 10, 64)
		m.Delta = &res
	} else {
		err = fmt.Errorf("wrong metric type %v", mt)
	}

	return err
}
