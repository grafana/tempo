package frontend

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

var copyHeaders = []string{
	"Authorization",
	"X-Scope-OrgID",
}

func headersFromGrpcContext(ctx context.Context) (hs http.Header) {
	hs = http.Header{}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// Tests may not have metadata so skip
		return
	}

	for _, h := range copyHeaders {
		if v := md.Get(h); len(v) > 0 {
			hs[h] = v
		}
	}

	return
}
