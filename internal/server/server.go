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

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"golang.org/x/sync/errgroup"
)

type MetricsServer interface {
	Run() error
	Stop(context.Context) error
}

type Server struct {
	metricsServer     MetricsServer
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

	if config.WorkMode == common.ModeGRPC {
		s.metricsServer, err = NewServerGRPC(config, logger, s.metricsStorage, s.storageSaver)
	} else if config.WorkMode == common.ModeHTTP {
		s.metricsServer, err = NewServerHTTP(config, logger, s.metricsStorage, s.storageSaver)
	} else {
		err = fmt.Errorf("unknown work mode: %v", config.WorkMode)
	}

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Server) Run() error {
	s.config.Print(s.logger)

	start := func() error {
		return s.metricsServer.Run()
	}

	stop := func() error {
		err := s.metricsServer.Stop(context.Background())
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
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
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
