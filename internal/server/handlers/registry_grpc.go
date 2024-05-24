package handlers

import (
	"context"

	"github.com/fuzzy-toozy/metrics-service/internal/grpcconv"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"github.com/fuzzy-toozy/metrics-service/internal/server/service"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type MetricsRegistryGRPC struct {
	serv         service.MetricsService
	log          log.Logger
	storageSaver storage.StorageSaver
	pb.UnimplementedMetricsServiceServer
}

var _ pb.MetricsServiceServer = (*MetricsRegistryGRPC)(nil)

func NewMetricsRegistryGRPC(serv service.MetricsService, storageSaver storage.StorageSaver, logger log.Logger) *MetricsRegistryGRPC {
	return &MetricsRegistryGRPC{
		serv:         serv,
		storageSaver: storageSaver,
		log:          logger,
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

func (r *MetricsRegistryGRPC) UpdateMetric(ctx context.Context, req *pb.UpdateRequest) (*pb.Metric, error) {
	reqM := &pb.Metric{}
	if err := proto.Unmarshal(req.GetData(), reqM); err != nil {
		r.log.Debugf("Failed to unmarshal update request: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	m := grpcconv.GRPCToMetric(reqM)
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

	if r.storageSaver != nil {
		err := r.storageSaver.Save()
		if err != nil {
			r.log.Errorf("Failed to update persistent storage: %v", err)
		}
	}

	return grpcconv.MetricToGRPC(mResp), nil
}

func (r *MetricsRegistryGRPC) UpdateMetrics(ctx context.Context, req *pb.UpdateRequest) (*pb.Metrics, error) {
	reqMetrics := &pb.Metrics{}
	if err := proto.Unmarshal(req.GetData(), reqMetrics); err != nil {
		r.log.Debugf("Failed to unmarshal update request: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	sMetrics := make([]metrics.Metric, 0, len(reqMetrics.Metrics))
	for _, reqM := range reqMetrics.Metrics {
		sMetrics = append(sMetrics, grpcconv.GRPCToMetric(reqM))
	}

	if err := r.serv.UpdateMetrics(sMetrics); err != nil {
		r.log.Debugf("Failed to update metrics: %v", err)
		return nil, status.Error(codes.Code(err.Code()), err.Error())
	}

	for i, m := range sMetrics {
		reqMetrics.Metrics[i] = grpcconv.MetricToGRPC(m)
	}

	if r.storageSaver != nil {
		err := r.storageSaver.Save()
		if err != nil {
			r.log.Errorf("Failed to update persistent storage: %v", err)
		}
	}

	return reqMetrics, nil
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
