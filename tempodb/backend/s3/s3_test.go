package s3

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

const getMethod = "GET"
const putMethod = "PUT"
const tagHeader = "X-Amz-Tagging"
const storageClassHeader = "X-Amz-Storage-Class"

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

			_ = r.ReadRange(ctx, "object", backend.KeyPath{"test"}, 10, []byte{}, false)
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

func fakeServerWithHeader(t *testing.T, obj *url.Values, testedHeaderName string) *httptest.Server {
	require.NotNil(t, obj)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case putMethod:
			// https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html
			switch testedHeaderValue := r.Header.Get(testedHeaderName); testedHeaderValue {
			case "":
			default:

				value, err := url.ParseQuery(testedHeaderValue)
				require.NoError(t, err)
				*obj = value
			}
		case getMethod:
			// return fake list response b/c it's the only call that has to succeed
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
		<ListBucketResult>
		</ListBucketResult>`))
		}

	}))
	t.Cleanup(server.Close)

	return server
}

func TestObjectBlockTags(t *testing.T) {

	tests := []struct {
		name string
		tags map[string]string
		// expectedObject raw.Object
	}{
		{
			"env", map[string]string{"env": "prod", "app": "thing"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// rawObject := raw.Object{}
			var obj url.Values

			server := fakeServerWithHeader(t, &obj, tagHeader)
			_, w, _, err := New(&Config{
				Region:    "blerg",
				AccessKey: "test",
				SecretKey: flagext.SecretWithValue("test"),
				Bucket:    "blerg",
				Insecure:  true,
				Endpoint:  server.URL[7:], // [7:] -> strip http://
				Tags:      tc.tags,
			})
			require.NoError(t, err)

			ctx := context.Background()
			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, false)

			for k, v := range tc.tags {
				vv := obj.Get(k)
				require.NotEmpty(t, vv)
				require.Equal(t, v, vv)
			}
		})
	}
}

func TestObjectWithPrefix(t *testing.T) {

	tests := []struct {
		name        string
		prefix      string
		objectName  string
		keyPath     backend.KeyPath
		httpHandler func(t *testing.T) http.HandlerFunc
	}{
		{
			name:       "with prefix",
			prefix:     "test_storage",
			objectName: "object",
			keyPath:    backend.KeyPath{"test"},
			httpHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.Method == getMethod {
						assert.Equal(t, r.URL.Query().Get("prefix"), "test_storage")

						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<ListBucketResult>
						</ListBucketResult>`))
						return
					}

					assert.Equal(t, r.URL.String(), "/blerg/test_storage/test/object")
				}
			},
		},
		{
			name:       "without prefix",
			prefix:     "",
			objectName: "object",
			keyPath:    backend.KeyPath{"test"},
			httpHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.Method == getMethod {
						assert.Equal(t, r.URL.Query().Get("prefix"), "")

						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<ListBucketResult>
						</ListBucketResult>`))
						return
					}

					assert.Equal(t, r.URL.String(), "/blerg/test/object")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testServer(t, tc.httpHandler(t))
			_, w, _, err := New(&Config{
				Region:    "blerg",
				AccessKey: "test",
				SecretKey: flagext.SecretWithValue("test"),
				Bucket:    "blerg",
				Prefix:    tc.prefix,
				Insecure:  true,
				Endpoint:  server.URL[7:],
			})
			require.NoError(t, err)

			ctx := context.Background()
			err = w.Write(ctx, tc.objectName, tc.keyPath, bytes.NewReader([]byte{}), 0, false)
			assert.NoError(t, err)
		})
	}
}

func TestObjectStorageClass(t *testing.T) {

	tests := []struct {
		name         string
		StorageClass string
		// expectedObject raw.Object
	}{
		{
			"Standard", "STANDARD",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// rawObject := raw.Object{}
			var obj url.Values

			server := fakeServerWithHeader(t, &obj, storageClassHeader)
			_, w, _, err := New(&Config{
				Region:       "blerg",
				AccessKey:    "test",
				SecretKey:    flagext.SecretWithValue("test"),
				Bucket:       "blerg",
				Insecure:     true,
				Endpoint:     server.URL[7:], // [7:] -> strip http://
				StorageClass: tc.StorageClass,
			})
			require.NoError(t, err)

			ctx := context.Background()
			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, false)
			require.Equal(t, obj.Has(tc.StorageClass), true)
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
