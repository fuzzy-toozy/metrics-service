package handlers

import (
	"bytes"
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func WithEncryptionGRPC(log logging.Logger, key *rsa.PrivateKey) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		updReq, ok := req.(*pb.UpdateRequest)
		if !ok {
			return handler(ctx, req)
		}

		decBody, err := encryption.DecryptRequestBody(bytes.NewBuffer(updReq.Data), key)
		if err != nil {
			log.Errorf("Failed to decrypt request body: %v", err)
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("failed to decrypt request body: %v", err))
		}

		updReq.Data = decBody.Bytes()

		return handler(ctx, updReq)
	}
}
