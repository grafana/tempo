package azure

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"sync/atomic"
	"testing"
	"time"

	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentials(t *testing.T) {
	_, _, _, err := New(&Config{})
	require.Error(t, err)

	os.Setenv("AZURE_STORAGE_ACCOUNT", "testing")
	os.Setenv("AZURE_STORAGE_KEY", "dGVzdGluZwo=")

	defer os.Unsetenv("AZURE_STORAGE_ACCOUNT")
	defer os.Unsetenv("AZURE_STORAGE_KEY")

	count := int32(0)
	server := fakeServer(t, 1*time.Second, &count)

	_, _, _, err = New(&Config{
		Endpoint: server.URL[7:], // [7:] -> strip http://,
	})
	require.NoError(t, err)
}

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
				StorageAccountName: "testing",
				StorageAccountKey:  flagext.SecretWithValue("YQo="),
				MaxBuffers:         3,
				BufferSize:         1000,
				ContainerName:      "blerg",
				Endpoint:           server.URL[7:], // [7:] -> strip http://,
				HedgeRequestsAt:    tc.hedgeAt,
				HedgeRequestsUpTo:  2,
			})
			require.NoError(t, err)

			ctx := context.Background()

			// the first call on each client initiates an extra http request
			// clearing that here
			_, _, _ = r.Read(ctx, "object", backend.KeyPathForBlock(uuid.New(), "tenant"), false)
			time.Sleep(tc.returnIn)
			atomic.StoreInt32(&count, 0)

			// calls that should hedge
			_, _, _ = r.Read(ctx, "object", backend.KeyPathForBlock(uuid.New(), "tenant"), false)
			time.Sleep(tc.returnIn)
			assert.Equal(t, tc.expectedHedgedRequests*2, atomic.LoadInt32(&count)) // *2 b/c reads execute a HEAD and GET
			atomic.StoreInt32(&count, 0)

			// this panics with the garbage test setup. todo: make it not panic
			// _ = r.ReadRange(ctx, "object", uuid.New(), "tenant", 10, make([]byte, 100))
			// time.Sleep(tc.returnIn)
			// assert.Equal(t, tc.expectedHedgedRequests, atomic.LoadInt32(&count))
			// atomic.StoreInt32(&count, 0)

			// calls that should not hedge
			_, _ = r.List(ctx, backend.KeyPath{"test"})
			assert.Equal(t, int32(1), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			_ = w.Write(ctx, "object", backend.KeyPathForBlock(uuid.New(), "tenant"), bytes.NewReader(make([]byte, 10)), 10, false)
			// Write consists of two operations:
			// - Put Block operation
			//   https://docs.microsoft.com/en-us/rest/api/storageservices/put-block
			// - Put Block List operation
			//   https://docs.microsoft.com/en-us/rest/api/storageservices/put-block-list
			assert.Equal(t, int32(2), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)
		})
	}
}

func fakeServer(t *testing.T, returnIn time.Duration, counter *int32) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(returnIn)

		atomic.AddInt32(counter, 1)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestReadError(t *testing.T) {
	// confirm blobNotFoundError converts to ErrDoesNotExist
	blobNotFoundError := blobStorageError(string(blob.ServiceCodeBlobNotFound))
	err := readError(blobNotFoundError)
	require.Equal(t, backend.ErrDoesNotExist, err)

	// wrap blob not found error and confirm it still converts to ErrDoesNotExist
	wrappedBlobNotFoundError := errors.Wrap(blobNotFoundError, "wrap")
	err = readError(wrappedBlobNotFoundError)
	require.Equal(t, backend.ErrDoesNotExist, err)

	// rando error is not returned as ErrDoesNotExist
	randoError := errors.New("blerg")
	err = readError(randoError)
	require.NotEqual(t, backend.ErrDoesNotExist, err)

	// other azure error is not returned as ErrDoesNotExist
	otherAzureError := blobStorageError(string(blob.ServiceCodeInternalError))
	err = readError(otherAzureError)
	require.NotEqual(t, backend.ErrDoesNotExist, err)
}

func blobStorageError(serviceCode string) error {
	resp := &http.Response{
		Header: http.Header{
			textproto.CanonicalMIMEHeaderKey("x-ms-error-code"): []string{serviceCode},
		},
		Request: httptest.NewRequest("GET", "/blobby/blob", nil), // azure error handling code will panic if Request is unset
	}

	return blob.NewResponseError(nil, resp, "")
}
