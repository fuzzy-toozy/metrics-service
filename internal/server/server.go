package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"golang.org/x/sync/errgroup"
)

func NewDefaultHTTPServer(config config.Config, logger log.Logger, handler http.Handler) *http.Server {

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

func Run(start func() error, stop func() error) error {
	ctx, cancel := context.WithCancel(context.Background())

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
