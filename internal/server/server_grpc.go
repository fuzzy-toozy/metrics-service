package server

import (
	"context"
	"net"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/fuzzy-toozy/metrics-service/internal/server/service"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

type ServerGRPC struct {
	config     *config.Config
	serverGRPC *grpc.Server
	log        logging.Logger
	listener   net.Listener
}

var _ MetricsServer = (*ServerGRPC)(nil)

func NewServerGRPC(config *config.Config, logger logging.Logger, metricsStorage storage.Repository, storageSaver storage.StorageSaver) (*ServerGRPC, error) {
	registry := handlers.NewMetricsRegistryGRPC(service.NewCommonMetricsServiceGRPC(metricsStorage), storageSaver, logger)

	listener, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		return nil, err
	}

	ic := make([]grpc.UnaryServerInterceptor, 0)

	ic = append(ic, handlers.WithLoggingGRPC(logger))

	if config.TrustedSubnetAddr != nil {
		ic = append(ic, handlers.WithSubnetFilterGRPC(logger, config.TrustedSubnetAddr))
	}

	if config.EncryptPrivKey != nil {
		ic = append(ic, handlers.WithEncryptionGRPC(logger, config.EncryptPrivKey))
	}

	if len(config.SecretKey) > 0 {
		ic = append(ic, handlers.WithSignatureGRPC(logger, config.SecretKey))
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(int(config.MaxBodySize)), grpc.ChainUnaryInterceptor(ic...))

	pb.RegisterMetricsServiceServer(s, registry)

	return &ServerGRPC{
		config:     config,
		log:        logger,
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
