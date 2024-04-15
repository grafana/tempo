package interceptor

import (
	"context"
	"strings"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

const streamingQuerierPrefix = "/tempopb.StreamingQuerier/"

func NewFrontendAPIUnaryTimeout(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, streamingQuerierPrefix) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		return handler(ctx, req)
	}
}

func NewFrontendAPIStreamTimeout(timeout time.Duration) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		if strings.HasPrefix(info.FullMethod, streamingQuerierPrefix) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ss.Context(), timeout)
			defer cancel()
		}

		return handler(srv, &grpc_middleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: ctx,
		})
	}
}
