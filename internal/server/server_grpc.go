package server

import (
	"context"
	"fmt"
	"net"

	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers/mgrpc"
	"github.com/fuzzy-toozy/metrics-service/internal/server/service"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/grpclog"
)

type ServerGRPC struct {
	config     *config.Config
	serverGRPC *grpc.Server
	log        logging.Logger
	listener   net.Listener
}

var _ MetricsServer = (*ServerGRPC)(nil)

func NewServerGRPC(config *config.Config, logger *zap.Logger, metricsStorage storage.Repository) (*ServerGRPC, error) {
	registry := mgrpc.NewMetricsRegistryGRPC(service.NewCommonMetricsServiceGRPC(metricsStorage), logger.Sugar())

	listener, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		return nil, err
	}

	var ic []grpc.UnaryServerInterceptor
	var opts []grpc.ServerOption

	grpclog.SetLoggerV2(zapgrpc.NewLogger(logger))

	ic = append(ic, mgrpc.WithLoggingGRPC(logger.Sugar()))

	if config.TrustedSubnetAddr != nil {
		ic = append(ic, mgrpc.WithSubnetFilterGRPC(logger.Sugar(), config.TrustedSubnetAddr))
	}

	if len(config.CaCertPath) > 0 {
		creds, err := encryption.SetupServerTLS(config.CaCertPath, config.EncKeyPath, config.ServerCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to setup TLS: %w", err)
		}

		opts = append(opts, grpc.Creds(creds))
	}

	opts = append(opts, grpc.MaxRecvMsgSize(int(config.MaxBodySize)))
	opts = append(opts, grpc.ChainUnaryInterceptor(ic...))

	s := grpc.NewServer(opts...)

	pb.RegisterMetricsServiceServer(s, registry)

	return &ServerGRPC{
		config:     config,
		log:        logger.Sugar(),
		serverGRPC: s,
		listener:   listener,
	}, nil
}

func (s *ServerGRPC) SetListener(l net.Listener) {
	s.listener = l
}

func (s *ServerGRPC) Run() error {
	return s.serverGRPC.Serve(s.listener)
}

func (s *ServerGRPC) Stop(ctx context.Context) error {
	s.serverGRPC.GracefulStop()
	return nil
}
