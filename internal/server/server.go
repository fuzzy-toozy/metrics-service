// Package server Metrics serving and storing server.
// Stores and serves metrics sent by agent.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	httpServer        *http.Server
	asyncStorageSaver *storage.PeriodicSaver
	storageSaver      storage.StorageSaver
	metricsStorage    storage.Repository
	config            *config.Config
	logger            logging.Logger
	stopCtx           context.Context
	stop              context.CancelFunc
}

func NewServer(logger logging.Logger) (*Server, error) {
	s := Server{}
	ctx, cancel := context.WithCancel(context.Background())
	s.stopCtx = ctx
	s.stop = cancel

	s.logger = logger
	config, err := config.BuildConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build server config: %w", err)
	}

	s.config = config

	if config.DatabaseConfig.UseDatabase {
		s.metricsStorage, err = storage.NewPGMetricRepository(config.DatabaseConfig, storage.NewDefaultDBRetryExecutor(s.stopCtx), s.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics storage: %w", err)
		}
	} else {
		s.metricsStorage = storage.NewCommonMetricsRepository()
	}

	registryHandler, err := handlers.NewDefaultMetricRegistryHandler(logger, s.metricsStorage, s.storageSaver, config.DatabaseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	if config.RestoreData && !config.DatabaseConfig.UseDatabase {
		const perms = 0444
		f, err := os.OpenFile(config.StoreFilePath, os.O_RDONLY, perms)
		if err == nil {
			err = s.metricsStorage.Load(f)
		}
		if err != nil {
			logger.Errorf("Failed to restore data from persistent storage(%v): %v", config.StoreFilePath, err)
		} else {
			logger.Infof("Successfully loaded data from persistent storage %v", config.StoreFilePath)
		}
	}

	if len(config.StoreFilePath) > 0 && !config.DatabaseConfig.UseDatabase {
		fileSaver := storage.NewFileSaver(s.metricsStorage, config.StoreFilePath, logger)
		if config.StoreInterval.D > 0 {
			s.asyncStorageSaver = storage.NewPeriodicSaver(config.StoreInterval.D, logger, fileSaver)
			s.asyncStorageSaver.Run()
			logger.Infof("Async persistent storage saver is started")
		} else {
			s.storageSaver = fileSaver
			logger.Infof("Persistent storage will be updated synchronously")
		}
	}

	serverHandler := handlers.SetupRouting(registryHandler)

	if s.config.SecretKey != nil {
		serverHandler = handlers.WithSignatureCheck(serverHandler, logger, config.SecretKey)
	}

	if s.config.EncryptPrivKey != nil {
		serverHandler = handlers.WithDecryption(serverHandler, s.config.EncryptPrivKey, logger)
	}

	serverHandler = handlers.WithCompression(serverHandler, logger)

	serverHandler = handlers.WithBodySizeLimit(serverHandler, config.MaxBodySize)
	serverHandler = handlers.WithLogging(serverHandler, logger)

	s.httpServer = NewDefaultHTTPServer(*config, logger, serverHandler)

	return &s, nil
}

func (s *Server) Run() error {
	s.config.Print(s.logger)

	start := func() error {
		return s.httpServer.ListenAndServe()
	}

	stop := func() error {
		err := s.httpServer.Shutdown(context.Background())
		if err != nil {
			s.logger.Errorf("server shutdown failed: %w", err)
		}

		if s.asyncStorageSaver != nil {
			s.asyncStorageSaver.Stop()
		}

		if err = s.metricsStorage.Close(); err != nil {
			s.logger.Errorf("Failed to close metrics storage: %v", err)
		}

		return err
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-c
		s.stop()
	}()

	g, gCtx := errgroup.WithContext(s.stopCtx)

	g.Go(func() error {
		return start()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return stop()
	})

	return g.Wait()
}

func (s *Server) Stop() {
	s.stop()
}

func NewDefaultHTTPServer(config config.Config, logger logging.Logger, handler http.Handler) *http.Server {

	s := http.Server{
		Addr:         config.ServerAddress,
		ReadTimeout:  config.ReadTimeout.D,
		WriteTimeout: config.WriteTimeout.D,
		IdleTimeout:  config.IdleTimeout.D,
		Handler:      handler,
	}

	logger.Infof("Server listens to: %v", config.ServerAddress)

	return &s
}
