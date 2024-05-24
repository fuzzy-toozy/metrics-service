package server

import (
	"context"
	"fmt"
	"net/http"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/fuzzy-toozy/metrics-service/internal/server/service"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
)

type ServerHTTP struct {
	config     *config.Config
	serverHTTP *http.Server
	log        logging.Logger
}

var _ MetricsServer = (*ServerHTTP)(nil)

func (s *ServerHTTP) Run() error {
	return s.serverHTTP.ListenAndServe()
}

func (s *ServerHTTP) Stop(ctx context.Context) error {
	return s.serverHTTP.Shutdown(ctx)
}

func NewServerHTTP(config *config.Config, logger logging.Logger, metricsStorage storage.Repository, storageSaver storage.StorageSaver) (*ServerHTTP, error) {

	s := &ServerHTTP{
		config: config,
		log:    logger,
	}

	registryHandler, err := handlers.NewDefaultMetricRegistryHandler(logger, service.NewCommonMetricsServiceHTTP(metricsStorage), storageSaver)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	serverHandler := handlers.SetupRouting(registryHandler)

	if s.config.SecretKey != nil {
		serverHandler = handlers.WithSignatureCheck(serverHandler, logger, config.SecretKey)
	}

	if s.config.EncryptPrivKey != nil {
		serverHandler = handlers.WithDecryption(serverHandler, logger, s.config.EncryptPrivKey)
	}

	serverHandler = handlers.WithCompression(serverHandler, logger)

	serverHandler = handlers.WithBodySizeLimit(serverHandler, config.MaxBodySize)

	if s.config.TrustedSubnetAddr != nil {
		serverHandler = handlers.WithSubnetFilter(serverHandler, logger, s.config.TrustedSubnetAddr)
	}

	serverHandler = handlers.WithLogging(serverHandler, logger)

	s.serverHTTP = &http.Server{
		Addr:         config.ServerAddress,
		ReadTimeout:  config.ReadTimeout.D,
		WriteTimeout: config.WriteTimeout.D,
		IdleTimeout:  config.IdleTimeout.D,
		Handler:      serverHandler,
	}

	logger.Infof("Server listens to: %v", config.ServerAddress)

	return s, nil
}
