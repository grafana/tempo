// Provenance-includes-location: https://github.com/weaveworks/common/blob/main/httpgrpc/httpgrpc.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: Weaveworks Ltd.

package httpgrpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kit/log/level"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"

	spb "github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/types"
	"github.com/gogo/status"

	"github.com/grafana/dskit/log"
)

// Errorf returns a HTTP gRPC error than is correctly forwarded over
// gRPC, and can eventually be converted back to a HTTP response with
// HTTPResponseFromError.
func Errorf(code int, tmpl string, args ...interface{}) error {
	return ErrorFromHTTPResponse(&HTTPResponse{
		Code: int32(code),
		Body: []byte(fmt.Sprintf(tmpl, args...)),
	})
}

// ErrorFromHTTPResponse converts an HTTP response into a grpc error
func ErrorFromHTTPResponse(resp *HTTPResponse) error {
	a, err := types.MarshalAny(resp)
	if err != nil {
		return err
	}

	return status.ErrorProto(&spb.Status{
		Code:    resp.Code,
		Message: string(resp.Body),
		Details: []*types.Any{a},
	})
}

// HTTPResponseFromError converts a grpc error into an HTTP response
func HTTPResponseFromError(err error) (*HTTPResponse, bool) {
	s, ok := statusFromError(err)
	if !ok {
		return nil, false
	}

	status := s.Proto()
	if len(status.Details) != 1 {
		return nil, false
	}

	var resp HTTPResponse
	if err := types.UnmarshalAny(status.Details[0], &resp); err != nil {
		level.Error(log.Global()).Log("msg", "got error containing non-response", "err", err)
		return nil, false
	}

	return &resp, true
}

// statusFromError tries to cast the given error into status.Status.
// If the given error, or any error from its tree are a status.Status,
// that status.Status and the outcome true are returned.
// Otherwise, nil and the outcome false are returned.
// This implementation differs from status.FromError() because the
// latter checks only if the given error can be cast to status.Status,
// and doesn't check other errors in the given error's tree.
func statusFromError(err error) (*status.Status, bool) {
	if err == nil {
		return nil, false
	}
	type grpcStatus interface{ GRPCStatus() *grpcstatus.Status }
	var gs grpcStatus
	if errors.As(err, &gs) {
		st := gs.GRPCStatus()
		if st == nil {
			return nil, false
		}
		return status.FromGRPCStatus(st), true
	}
	return nil, false
}

const (
	MetadataMethod = "httpgrpc-method"
	MetadataURL    = "httpgrpc-url"
)

// AppendRequestMetadataToContext appends metadata of HTTPRequest into gRPC metadata.
func AppendRequestMetadataToContext(ctx context.Context, req *HTTPRequest) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		MetadataMethod, req.Method,
		MetadataURL, req.Url)
}
