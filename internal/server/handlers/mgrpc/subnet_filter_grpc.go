package mgrpc

import (
	"context"
	"fmt"
	"net"

	"github.com/fuzzy-toozy/metrics-service/internal/common"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func WithSubnetFilterGRPC(log logging.Logger, subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		var ipAddr string
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get(common.IPAddrKey)
			if len(values) > 0 {
				ipAddr = values[0]
			}
		}

		if len(ipAddr) == 0 {
			log.Debugf("unable to find ip address in incoming context for key %v", common.IPAddrKey)
			return nil, status.Error(codes.PermissionDenied, "unable to find ip address in incoming context")
		}

		parsedIP := net.ParseIP(ipAddr)

		if parsedIP == nil {
			log.Debugf("unable to parse ip address from incoming context for key %v", common.IPAddrKey)
			return nil, status.Error(codes.PermissionDenied, "unable to find parse ip address in incoming context")
		}

		maskedIP := parsedIP.Mask(subnet.Mask)
		if !maskedIP.Equal(subnet.IP) {
			respS := fmt.Sprintf("cient IP address is not in trusted subnet. Address: %s. Trusted Subnet: %s", ipAddr, subnet.String())
			log.Debugf(respS)
			return nil, status.Error(codes.PermissionDenied, respS)
		}

		return handler(ctx, req)
	}
}
