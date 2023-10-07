package common

import "strconv"

const MetricTypeGauge = "gauge"

const MetricTypeCounter = "counter"

type Float struct {
	Val float64
}

type Int struct {
	Val int64
}

func (m Float) GetValue() string {
	return strconv.FormatFloat(m.Val, 'f', -1, 64)
}

func (m Int) GetValue() string {
	return strconv.FormatInt(m.Val, 10)
}
