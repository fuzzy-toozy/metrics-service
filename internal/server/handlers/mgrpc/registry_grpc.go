package mgrpc

import (
	"context"

	"github.com/fuzzy-toozy/metrics-service/internal/grpcconv"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"github.com/fuzzy-toozy/metrics-service/internal/server/service"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricsRegistryGRPC struct {
	serv service.MetricsService
	log  log.Logger
	pb.UnimplementedMetricsServiceServer
}

var _ pb.MetricsServiceServer = (*MetricsRegistryGRPC)(nil)

func NewMetricsRegistryGRPC(serv service.MetricsService, logger log.Logger) *MetricsRegistryGRPC {
	return &MetricsRegistryGRPC{
		serv: serv,
		log:  logger,
	}
}

func (r *MetricsRegistryGRPC) GetMetric(ctx context.Context, req *pb.MetricRequest) (*pb.Metric, error) {
	m, err := r.serv.GetMetric(req.Id, req.Mtype)
	if err != nil {
		r.log.Debugf("Failed to get metric[ID: %s, Type: %s]: %v", req.Id, req.Mtype, err)
		return nil, status.Error(codes.Code(err.Code()), err.Error())
	}

	return grpcconv.MetricToGRPC(m), nil
}

func (r *MetricsRegistryGRPC) UpdateMetric(ctx context.Context, req *pb.MetricUpdateRequest) (*pb.Metric, error) {
	m := grpcconv.GRPCToMetric(req.Metric)
	val, err := m.GetData()
	if err != nil {
		r.log.Debugf("Failed to get metric value: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	mResp, errS := r.serv.UpdateMetric(m.MType, m.ID, val)
	if errS != nil {
		r.log.Debugf("Failed to update metric: %v", errS)
		return nil, status.Error(codes.Code(errS.Code()), errS.Error())
	}

	return grpcconv.MetricToGRPC(mResp), nil
}

func (r *MetricsRegistryGRPC) UpdateMetrics(ctx context.Context, req *pb.MetricsUpdateRequest) (*pb.Metrics, error) {

	sMetrics := grpcconv.GRPCToMetrics(req.Metrics)
	res, err := r.serv.UpdateMetrics(sMetrics)
	if err != nil {
		r.log.Debugf("Failed to update metrics: %v", err)
		return nil, status.Error(codes.Code(err.Code()), err.Error())
	}

	resp := grpcconv.MetricsToGRPC(res)

	return &pb.Metrics{Metrics: resp}, nil
}

func (r *MetricsRegistryGRPC) GetAllMetrics(ctx context.Context, e *empty.Empty) (*pb.Metrics, error) {
	m, err := r.serv.GetAllMetrics()
	if err != nil {
		r.log.Debugf("Failed to get metrics: %v", err)
		return nil, status.Error(codes.Code(err.Code()), err.Error())
	}

	respM := &pb.Metrics{}
	respM.Metrics = make([]*pb.Metric, 0, len(m))
	for _, metric := range m {
		respM.Metrics = append(respM.Metrics, grpcconv.MetricToGRPC(metric))
	}

	return respM, nil
}
