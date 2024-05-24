package handlers

import (
	"context"
	"fmt"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func WithSignatureGRPC(log logging.Logger, key []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		updReq, ok := req.(*pb.UpdateRequest)
		if !ok {
			return handler(ctx, req)
		}

		var sigHash string
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get(common.SighashKey)
			if len(values) > 0 {
				sigHash = values[0]
			}
		}

		if len(sigHash) == 0 {
			log.Debugf("unable to find data signature in incoming contesxt")
			return nil, status.Error(codes.PermissionDenied, "unable to find data signature in incoming contesxt")
		}

		err = encryption.CheckData(updReq.Data, key, sigHash)
		if err != nil {
			log.Debugf("Failed to validate body signature: %v", err)
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Failed to validate body signature: %v", err))
		}

		return handler(ctx, req)
	}
}
