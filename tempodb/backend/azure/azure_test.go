package azure

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

func TestCredentials(t *testing.T) {
	_, _, _, err := New(&Config{})
	require.Error(t, err)

	os.Setenv("AZURE_STORAGE_ACCOUNT", "testing")
	os.Setenv("AZURE_STORAGE_KEY", "dGVzdGluZwo=")

	defer os.Unsetenv("AZURE_STORAGE_ACCOUNT")
	defer os.Unsetenv("AZURE_STORAGE_KEY")

	count := int32(0)
	server := fakeServer(t, 1*time.Second, &count, nil)

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
			// The Azure SDK returns an error if write requests don't return 201.
			// We hardcode the UUID of the written block to ensure a consistent path on which to match.
			codes := map[string]int{
				"/testing/blerg/tenant/f97223f3-d60c-4923-b255-bb7b8140b389/object": http.StatusCreated,
			}
			server := fakeServer(t, tc.returnIn, &count, codes)

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

			blockSize := 2000000
			u, err := uuid.Parse("f97223f3-d60c-4923-b255-bb7b8140b389")
			require.NoError(t, err)
			_ = w.Write(ctx, "object", backend.KeyPathForBlock(u, "tenant"), bytes.NewReader(make([]byte, blockSize)), 10, false)
			// Write consists of two operations:
			// - Put Block operation
			//   https://docs.microsoft.com/en-us/rest/api/storageservices/put-block
			// - Put Block List operation
			//   https://docs.microsoft.com/en-us/rest/api/storageservices/put-block-list
			//
			// If the written bytes can fit in a single block, the Azure SDK will not call Put Block List, and will
			// instead perform a single upload.
			// In order to more closely resemble a real-world upload scenario, and to force the SDK to call
			// Put Block List, we deliberately create a payload that exceeds the size of a single block, forcing the
			// SDK to make three requests in total: one request for each of the two blocks, and a final commit request.
			// See azblob.UploadStreamOptions.BlockSize.
			assert.Equal(t, int32(3), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)
		})
	}
}

func fakeServer(t *testing.T, returnIn time.Duration, counter *int32, codes map[string]int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(returnIn)

		// If this path requires a certain response code, set that here.
		if code, ok := codes[r.URL.Path]; ok {
			w.WriteHeader(code)
		}

		atomic.AddInt32(counter, 1)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestReadError(t *testing.T) {
	// confirm blobNotFoundError converts to ErrDoesNotExist
	blobNotFoundError := blobStorageError(string(bloberror.BlobNotFound))
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
	otherAzureError := blobStorageError(string(bloberror.InternalError))
	err = readError(otherAzureError)
	require.NotEqual(t, backend.ErrDoesNotExist, err)
}

func blobStorageError(serviceCode string) error {
	resp := &http.Response{
		Header: http.Header{
			textproto.CanonicalMIMEHeaderKey("x-ms-error-code"): []string{serviceCode},
		},
		// We need to set both Request and Body to prevent the Azure SDK from panicking.
		Request: httptest.NewRequest("GET", "/blobby/blob", nil),
		Body:    io.NopCloser(bytes.NewReader([]byte("some response"))),
	}

	return runtime.NewResponseError(resp)
}

func TestObjectWithPrefix(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		objectName   string
		keyPath      backend.KeyPath
		expectedPath string
	}{
		{
			name:         "with prefix",
			prefix:       "test_prefix",
			objectName:   "object",
			keyPath:      backend.KeyPath{"test_path"},
			expectedPath: "/testing_account/blerg/test_prefix/test_path/object",
		},
		{
			name:         "without prefix",
			prefix:       "",
			objectName:   "object",
			keyPath:      backend.KeyPath{"test_path"},
			expectedPath: "/testing_account/blerg/test_path/object",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Handle the initial call to get container properties.
				if r.Method == "GET" {
					_, _ = w.Write([]byte(""))
					return
				}

				// The Azure SDK expects that successful write requests return 201, and will error if they don't.
				w.WriteHeader(http.StatusCreated)
				assert.Equal(t, tc.expectedPath, r.URL.Path)
			})
			_, w, _, err := New(&Config{
				StorageAccountName: "testing_account",
				StorageAccountKey:  flagext.SecretWithValue("YQo="),
				MaxBuffers:         3,
				BufferSize:         1000,
				ContainerName:      "blerg",
				Prefix:             tc.prefix,
				Endpoint:           server.URL[7:], // [7:] -> strip http://,
			})
			require.NoError(t, err)

			ctx := context.Background()
			err = w.Write(ctx, tc.objectName, tc.keyPath, bytes.NewReader([]byte{}), 0, false)
			assert.NoError(t, err)
		})
	}
}

func testServer(t *testing.T, httpHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	assert.NotNil(t, httpHandler)
	server := httptest.NewServer(httpHandler)
	t.Cleanup(server.Close)
	return server
}
