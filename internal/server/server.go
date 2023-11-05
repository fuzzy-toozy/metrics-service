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
	"github.com/fuzzy-toozy/metrics-service/internal/server/routing"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	httpServer        *http.Server
	asyncStorageSaver *storage.PeriodicSaver
	storageSaver      storage.StorageSaver
	metricsStorage    storage.MetricsStorage
	config            *config.Config
	logger            logging.Logger
}

func NewServer(logger logging.Logger) (*Server, error) {
	s := Server{}
	s.logger = logger
	config, err := config.BuildConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build server config: %w", err)
	}

	s.config = config

	s.metricsStorage = storage.NewCommonMetricsStorage()
	registryHandler := handlers.NewDefaultMetricRegistryHandler(logger, s.metricsStorage, s.storageSaver, config.DatabaseConfig)

	if config.RestoreData {
		f, err := os.OpenFile(config.StoreFilePath, os.O_RDONLY, 0444)
		if err == nil {
			err = s.metricsStorage.Load(f)
		}
		if err != nil {
			logger.Errorf("Failed to restore data from persistent storage(%v): %v", config.StoreFilePath, err)
		} else {
			logger.Infof("Successfully loaded data from persistent storage %v", config.StoreFilePath)
		}
	}

	if len(config.StoreFilePath) > 0 {
		fileSaver := storage.NewFileSaver(s.metricsStorage, config.StoreFilePath, logger)
		if config.StoreInterval > 0 {
			s.asyncStorageSaver = storage.NewPeriodicSaver(config.StoreInterval, logger, fileSaver)
			s.asyncStorageSaver.Run()
			logger.Infof("Async persistent storage saver is started")
		} else {
			s.storageSaver = fileSaver
			logger.Infof("Persistent storage will be updated synchronously")
		}
	}

	routerHandler := routing.SetupRouting(registryHandler)
	serverHandler := handlers.WithLogging(
		handlers.WithCompression(routerHandler, logger),
		logger)

	s.httpServer = NewDefaultHTTPServer(*config, logger, serverHandler)

	return &s, nil
}

func (s *Server) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	start := func() error {
		return s.httpServer.ListenAndServe()
	}

	stop := func() error {
		err := s.httpServer.Shutdown(context.Background())
		if err != nil {
			err = fmt.Errorf("erver shutdown failed: %w", err)
		}

		if s.asyncStorageSaver != nil {
			s.asyncStorageSaver.Stop()
		}
		return err
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-c
		cancel()
	}()

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return start()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return stop()
	})

	return g.Wait()
}

func NewDefaultHTTPServer(config config.Config, logger logging.Logger, handler http.Handler) *http.Server {

	s := http.Server{
		Addr:         config.ServerAddress,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
		Handler:      handler,
	}

	logger.Infof("Server listens to: %v", config.ServerAddress)

	return &s
}
