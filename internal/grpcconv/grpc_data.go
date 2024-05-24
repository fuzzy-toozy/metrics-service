package grpcconv

import (
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
)

func MetricToGRPC(m metrics.Metric) *pb.Metric {
	pbM := &pb.Metric{Mtype: m.MType, Id: m.ID}
	if m.Delta != nil {
		pbM.Delta = *m.Delta
	} else if m.Value != nil {
		pbM.Value = *m.Value
	}

	return pbM
}

func GRPCToMetric(m *pb.Metric) metrics.Metric {
	mM := metrics.Metric{
		MType: m.Mtype,
		ID:    m.Id,
	}

	if m.Mtype == metrics.GaugeMetricType {
		mM.Value = &m.Value
	} else if m.Mtype == metrics.CounterMetricType {
		mM.Delta = &m.Delta
	}

	return mM
}
