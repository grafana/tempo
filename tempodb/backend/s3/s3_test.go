package s3

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHedge(t *testing.T) {
	tests := []struct {
		name                   string
		returnIn               time.Duration
		hedgeAt                time.Duration
		expectedHedgedRequests int32
	}{
		{
			name:                   "hedge disabled",
			expectedHedgedRequests: 1,
		},
		{
			name:                   "hedge enabled doesn't hit",
			hedgeAt:                time.Hour,
			expectedHedgedRequests: 1,
		},
		{
			name:                   "hedge enabled and hits",
			hedgeAt:                time.Millisecond,
			returnIn:               100 * time.Millisecond,
			expectedHedgedRequests: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			count := int32(0)
			server := fakeServer(t, tc.returnIn, &count)

			r, w, _, err := New(&Config{
				Region:            "blerg",
				AccessKey:         "test",
				SecretKey:         flagext.SecretWithValue("test"),
				Bucket:            "blerg",
				Insecure:          true,
				Endpoint:          server.URL[7:], // [7:] -> strip http://
				HedgeRequestsAt:   tc.hedgeAt,
				HedgeRequestsUpTo: 2,
			})
			require.NoError(t, err)

			ctx := context.Background()

			// the first call on each client initiates an extra http request
			// clearing that here
			_, _, _ = r.Read(ctx, "object", backend.KeyPath{"test"}, false)
			time.Sleep(tc.returnIn)
			atomic.StoreInt32(&count, 0)

			// calls that should hedge
			_, _, _ = r.Read(ctx, "object", backend.KeyPath{"test"}, false)
			time.Sleep(tc.returnIn)
			assert.Equal(t, tc.expectedHedgedRequests, atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			_ = r.ReadRange(ctx, "object", backend.KeyPath{"test"}, 10, []byte{})
			time.Sleep(tc.returnIn)
			assert.Equal(t, tc.expectedHedgedRequests, atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			// calls that should not hedge
			_, _ = r.List(ctx, backend.KeyPath{"test"})
			assert.Equal(t, int32(1), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, false)
			assert.Equal(t, int32(1), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)
		})
	}
}

func fakeServer(t *testing.T, returnIn time.Duration, counter *int32) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(returnIn)

		atomic.AddInt32(counter, 1)
		// return fake list response b/c it's the only call that has to succeed
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
		<ListBucketResult>
		</ListBucketResult>`))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestReadError(t *testing.T) {
	errA := minio.ErrorResponse{
		Code: s3.ErrCodeNoSuchKey,
	}
	errB := readError(errA)
	assert.Equal(t, backend.ErrDoesNotExist, errB)

	wups := fmt.Errorf("wups")
	errB = readError(wups)
	assert.Equal(t, wups, errB)
}
