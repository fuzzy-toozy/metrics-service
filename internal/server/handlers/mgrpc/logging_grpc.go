package mgrpc

import (
	"context"
	"time"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	"google.golang.org/grpc"
)

func WithLoggingGRPC(log logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()

		resp, err = handler(ctx, req)

		log.Debugf("Method: %v, ExecTime: %v, Err: %v", info.FullMethod, time.Since(start), err)

		return resp, err
	}
}
