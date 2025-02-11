package middleware

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	MetadataClusterKey = "x-cluster"
)

// ClusterUnaryClientInterceptor propagates the given cluster info to gRPC metadata.
func ClusterUnaryClientInterceptor(cluster string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if cluster != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, MetadataClusterKey, cluster)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ClusterUnaryServerInterceptor checks if the incoming gRPC metadata contains any cluster information and if so,
// checks if the latter corresponds to the given cluster. If it is the case, the request is further propagated.
// Otherwise, an error is returned.
func ClusterUnaryServerInterceptor(cluster string, logger log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		reqCluster := getClusterFromIncomingContext(ctx, logger)
		if cluster != reqCluster {
			msg := fmt.Sprintf("request intended for cluster %q - this is cluster %q", reqCluster, cluster)
			level.Warn(logger).Log("msg", msg)
			return nil, status.Error(codes.FailedPrecondition, msg)
		}
		return handler(ctx, req)
	}
}

func getClusterFromIncomingContext(ctx context.Context, logger log.Logger) string {
	clusterIDs := metadata.ValueFromIncomingContext(ctx, MetadataClusterKey)
	if len(clusterIDs) != 1 {
		msg := fmt.Sprintf("gRPC metadata should contain exactly 1 value for key \"%s\", but the current set of values is %v. Returning an empty string.", MetadataClusterKey, clusterIDs)
		level.Warn(logger).Log("msg", msg)
		return ""
	}
	return clusterIDs[0]
}
