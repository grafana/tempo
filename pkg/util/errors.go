package util

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrTraceNotFound can be used when we don't find a trace
	ErrTraceNotFound = errors.New("trace not found")

	// ErrSearchKeyValueNotFound is used to indicate the requested key/value pair was not found.
	ErrSearchKeyValueNotFound = errors.New("key/value not found")

	ErrUnsupported = fmt.Errorf("unsupported")
)

// IsConnCanceled returns true, if error is from a closed gRPC connection.
// copied from https://github.com/etcd-io/etcd/blob/7f47de84146bdc9225d2080ec8678ca8189a2d2b/clientv3/client.go#L646
func IsConnCanceled(err error) bool {
	if err == nil {
		return false
	}

	// >= gRPC v1.23.x
	s, ok := status.FromError(err)
	if ok {
		// connection is canceled or server has already closed the connection
		return s.Code() == codes.Canceled || s.Message() == "transport is closing"
	}

	return false
}
