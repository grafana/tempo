package app

import (
	"context"
	"net/http"

	"github.com/grafana/dskit/middleware"
	"github.com/grafana/dskit/user"
	"google.golang.org/grpc"
)

func fakeHTTPAuthMiddleware(tenantID string) middleware.Interface {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := user.InjectOrgID(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
}

func fakeGRPCAuthUnaryMiddleware(tenantID string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = user.InjectOrgID(ctx, tenantID)
		return handler(ctx, req)
	}
}

func fakeGRPCAuthStreamMiddleware(tenantID string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := user.InjectOrgID(ss.Context(), tenantID)
		return handler(srv, serverStream{
			ctx:          ctx,
			ServerStream: ss,
		})
	}
}

type serverStream struct {
	ctx context.Context
	grpc.ServerStream
}

func (ss serverStream) Context() context.Context {
	return ss.ctx
}
