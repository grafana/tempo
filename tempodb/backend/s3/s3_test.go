package s3

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafana/dskit/flagext"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

const (
	getMethod           = "GET"
	putMethod           = "PUT"
	tagHeader           = "X-Amz-Tagging"
	storageClassHeader  = "X-Amz-Storage-Class"
	sseHeader           = "X-Amz-Server-Side-Encryption"
	sseKMSKeyIDHeader   = "X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id"
	sseKMSContextHeader = "X-Amz-Server-Side-Encryption-Context"

	defaultAccessKey = "AKIAIOSFODNN7EXAMPLE"
	defaultSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	user1AccessKey   = "AKIAI44QH8DHBEXAMPLE"
	user1SecretKey   = "je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY"
)

type ec2RoleCredRespBody struct {
	Expiration      time.Time `json:"Expiration"`
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	Token           string    `json:"Token"`
	Code            string    `json:"Code"`
	Message         string    `json:"Message"`
	LastUpdated     time.Time `json:"LastUpdated"`
	Type            string    `json:"Type"`
}

// TestFetchCreds verifies that fetchCreds() correctly handles individual
// credential sources. Each test only configures the specific source being
// validated.
func TestFetchCreds(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Set up mock IAM endpoint once for all tests (for security and efficiency)
	metadataSrv := httptest.NewServer(metadataMockedHandler(t))
	t.Cleanup(metadataSrv.Close)

	tests := []struct {
		name     string
		access   string
		secret   string
		envs     map[string]string
		profile  string
		expected credentials.Value
	}{
		{
			name: "no-creds",
			// anonymous access is the last in the chain of fetchCreds,
			// so we need to set an invalid endpoint to prevent IAM access
			envs: map[string]string{
				"TEST_IAM_ENDPOINT": "http://invalid-endpoint-to-prevent-iam-access:9999",
			},
			expected: credentials.Value{
				SignerType: credentials.SignatureAnonymous,
			},
		},
		{
			name: "aws-env",
			envs: map[string]string{
				"AWS_ACCESS_KEY_ID":     defaultAccessKey,
				"AWS_SECRET_ACCESS_KEY": defaultSecretKey,
			},
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureV4,
			},
		},
		{
			name:   "aws-static",
			access: defaultAccessKey,
			secret: defaultSecretKey,
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureDefault,
			},
		},
		{
			name: "minio-env",
			envs: map[string]string{
				"MINIO_ACCESS_KEY": defaultAccessKey,
				"MINIO_SECRET_KEY": defaultSecretKey,
			},
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureV4,
			},
		},
		{
			name: "aws-config-no-profile",
			envs: map[string]string{
				"AWS_SHARED_CREDENTIALS_FILE": filepath.Join(cwd, "testdata/aws-credentials"),
			},
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureV4,
			},
		},
		{
			name: "aws-config-with-profile",
			envs: map[string]string{
				"AWS_SHARED_CREDENTIALS_FILE": filepath.Join(cwd, "testdata/aws-credentials"),
				"AWS_PROFILE":                 "user1",
			},
			expected: credentials.Value{
				AccessKeyID:     user1AccessKey,
				SecretAccessKey: user1SecretKey,
				SignerType:      credentials.SignatureV4,
			},
		},
		{
			name: "minio-config",
			envs: map[string]string{
				"MINIO_SHARED_CREDENTIALS_FILE": filepath.Join(cwd, "testdata/minio-config.json"),
				"MINIO_ALIAS":                   "s3",
			},
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureV4,
			},
		},
		{
			name: "aws-iam-irsa-mocked",
			envs: map[string]string{
				"AWS_WEB_IDENTITY_TOKEN_FILE": filepath.Join(cwd, "testdata/iam-token"),
				"AWS_ROLE_ARN":                "arn:aws:iam::123456789012:role/role-name",
				"AWS_ROLE_SESSION_NAME":       "tempo",
			},
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureV4,
			},
		},
		{
			name: "aws-iam-imds-mocked",
			envs: map[string]string{
				"AWS_ROLE_ARN": "arn:aws:iam::123456789012:role/role-name",
			},
			expected: credentials.Value{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SignerType:      credentials.SignatureV4,
				Expiration:      timeNow().Add(time.Hour),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all credential-related environment variables for isolation
			// because some may exist in the environment
			credentialVars := []string{
				"AWS_ACCESS_KEY_ID",
				"AWS_SECRET_ACCESS_KEY",
				"AWS_SESSION_TOKEN",
				"AWS_PROFILE",
				"AWS_SHARED_CREDENTIALS_FILE",
				"AWS_CONFIG_FILE",
				"MINIO_ACCESS_KEY",
				"MINIO_SECRET_KEY",
				"MINIO_SHARED_CREDENTIALS_FILE",
				"MINIO_ALIAS",
				"AWS_WEB_IDENTITY_TOKEN_FILE",
				"AWS_ROLE_ARN",
				"AWS_ROLE_SESSION_NAME",
			}
			for _, envVar := range credentialVars {
				os.Unsetenv(envVar)
			}

			// Use shared mock IAM endpoint for security (prevents real IAM access if it exists)
			t.Setenv("TEST_IAM_ENDPOINT", metadataSrv.URL)

			// Set test-specific environment variables
			for name, value := range tc.envs {
				t.Setenv(name, value)
			}

			c := &Config{}
			if tc.access != "" {
				c.AccessKey = tc.access
				c.SecretKey = flagext.SecretWithValue(tc.secret)
			}

			creds, err := fetchCreds(c)
			assert.NoError(t, err)

			realCreds, err := creds.GetWithContext(nil)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, realCreds)
		})
	}
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
			_, _, _ = r.Read(ctx, "object", backend.KeyPath{"test"}, nil)
			time.Sleep(tc.returnIn)
			atomic.StoreInt32(&count, 0)

			// calls that should hedge
			_, _, _ = r.Read(ctx, "object", backend.KeyPath{"test"}, nil)
			time.Sleep(tc.returnIn)
			assert.Equal(t, tc.expectedHedgedRequests, atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			_ = r.ReadRange(ctx, "object", backend.KeyPath{"test"}, 10, []byte{}, nil)
			time.Sleep(tc.returnIn)
			assert.Equal(t, tc.expectedHedgedRequests, atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			// calls that should not hedge
			_, _ = r.List(ctx, backend.KeyPath{"test"})
			assert.Equal(t, int32(1), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)

			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, nil)
			assert.Equal(t, int32(1), atomic.LoadInt32(&count))
			atomic.StoreInt32(&count, 0)
		})
	}
}

func TestRetryConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		retryMaxAttempts    int
		retryBackoffInitial time.Duration
		retryBackoffMax     time.Duration
	}{
		{
			name:                "custom retry configuration",
			retryMaxAttempts:    5,
			retryBackoffInitial: 500 * time.Millisecond,
			retryBackoffMax:     10 * time.Second,
		},
		{
			name:                "default retry values when not set",
			retryMaxAttempts:    0,
			retryBackoffInitial: 0,
			retryBackoffMax:     0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origMaxRetry := minio.MaxRetry
			origRetryUnit := minio.DefaultRetryUnit
			origRetryCap := minio.DefaultRetryCap
			defer func() {
				minio.MaxRetry = origMaxRetry
				minio.DefaultRetryUnit = origRetryUnit
				minio.DefaultRetryCap = origRetryCap
			}()

			server := fakeServer(t, 100*time.Millisecond, new(int32))

			cfg := &Config{
				Region:              "blerg",
				AccessKey:           "test",
				SecretKey:           flagext.SecretWithValue("test"),
				Bucket:              "blerg",
				Insecure:            true,
				Endpoint:            server.URL[7:],
				RetryMaxAttempts:    tc.retryMaxAttempts,
				RetryBackoffInitial: tc.retryBackoffInitial,
				RetryBackoffMax:     tc.retryBackoffMax,
			}

			_, _, _, err := New(cfg)
			require.NoError(t, err)

			if tc.retryMaxAttempts != 0 {
				assert.Equal(t, tc.retryMaxAttempts, minio.MaxRetry)
			} else {
				assert.Equal(t, origMaxRetry, minio.MaxRetry)
			}

			if tc.retryBackoffInitial != 0 {
				assert.Equal(t, tc.retryBackoffInitial, minio.DefaultRetryUnit)
			} else {
				assert.Equal(t, origRetryUnit, minio.DefaultRetryUnit)
			}

			if tc.retryBackoffMax != 0 {
				assert.Equal(t, tc.retryBackoffMax, minio.DefaultRetryCap)
			} else {
				assert.Equal(t, origRetryCap, minio.DefaultRetryCap)
			}
		})
	}
}

func TestNilConfig(t *testing.T) {
	_, _, _, err := New(nil)
	require.Error(t, err)

	_, _, _, err = NewNoConfirm(nil)
	require.Error(t, err)
}

func fakeServer(t *testing.T, returnIn time.Duration, counter *int32) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		Code: minio.NoSuchKey,
	}
	errB := readError(errA)
	assert.Equal(t, backend.ErrDoesNotExist, errB)

	wups := fmt.Errorf("wups")
	errB = readError(wups)
	assert.Equal(t, wups, errB)
}

func fakeServerWithHeader(t *testing.T, httpHeader *http.Header) *httptest.Server {
	require.NotNil(t, httpHeader)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case putMethod:
			*httpHeader = r.Header
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
			var httpHeaders http.Header

			server := fakeServerWithHeader(t, &httpHeaders)
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
			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, nil)

			testedHeaderValue := httpHeaders.Get(tagHeader)
			require.NotEmpty(t, testedHeaderValue)
			headerValue, err := url.ParseQuery(testedHeaderValue)
			require.NoError(t, err)

			for k, v := range tc.tags {
				vv := headerValue.Get(k)
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
						assert.Equal(t, "test_storage/", r.URL.Query().Get("prefix"))

						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<ListBucketResult>
						</ListBucketResult>`))
						return
					}

					assert.Equal(t, "/blerg/test_storage/test/object", r.URL.String())
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

					assert.Equal(t, "/blerg/test/object", r.URL.String())
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
			err = w.Write(ctx, tc.objectName, tc.keyPath, bytes.NewReader([]byte{}), 0, nil)
			assert.NoError(t, err)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		objectName  string
		keyPath     backend.KeyPath
		httpHandler func(t *testing.T) http.HandlerFunc
	}{
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
					assert.Equal(t, "/blerg/test/object", r.URL.String())
					w.WriteHeader(http.StatusNoContent)
				}
			},
		},
		{
			name:       "with prefix",
			prefix:     "test_storage",
			objectName: "object",
			keyPath:    backend.KeyPath{"test"},
			httpHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.Method == getMethod {
						assert.Equal(t, "test_storage/", r.URL.Query().Get("prefix"))

						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<ListBucketResult>
						</ListBucketResult>`))
						return
					}
					assert.Equal(t, "/blerg/test_storage/test/object", r.URL.String())
					w.WriteHeader(http.StatusNoContent)
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
			err = w.Delete(ctx, tc.objectName, tc.keyPath, nil)
			assert.NoError(t, err)
		})
	}
}

func TestListBlocksWithPrefix(t *testing.T) {
	tests := []struct {
		name              string
		prefix            string
		tenant            string
		liveBlockIDs      []uuid.UUID
		compactedBlockIDs []uuid.UUID
		noCompactBlockIDs []uuid.UUID
		httpHandler       func(t *testing.T) http.HandlerFunc
	}{
		{
			name:              "with prefix",
			prefix:            "a/b/c/",
			tenant:            "single-tenant",
			liveBlockIDs:      []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000000")},
			compactedBlockIDs: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001")},
			noCompactBlockIDs: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000000")},
			httpHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.Method == getMethod {
						assert.Equal(t, "a/b/c/single-tenant/", r.URL.Query().Get("prefix"))

						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<ListBucketResult>
							<Name>blerg</Name>
							<Prefix>a/b/c</Prefix>
							<ContinuationToken></ContinuationToken>
							<KeyCount>2</KeyCount>
							<MaxKeys>100</MaxKeys>
							<EncodingType>url</EncodingType>
							<IsTruncated>false</IsTruncated>
							<Contents>
								<Key>a/b/c/single-tenant/00000000-0000-0000-0000-000000000000/meta.json</Key>
								<LastModified>2024-03-01T00:00:00.000Z</LastModified>
								<ETag>&quot;d42a22ddd183f61924c661b1c026c1ef&quot;</ETag>
								<Size>398</Size>
								<StorageClass>STANDARD</StorageClass>
							</Contents>
							<Contents>
								<Key>a/b/c/single-tenant/00000000-0000-0000-0000-000000000000/nocompact.flg</Key>
								<LastModified>2024-03-01T00:00:00.000Z</LastModified>
								<ETag>&quot;d41d8cd98f00b204e9800998ecf8427e&quot;</ETag>
								<Size>0</Size>
								<StorageClass>STANDARD</StorageClass>
							</Contents>
							
							<Contents>
								<Key>a/b/c/single-tenant/00000000-0000-0000-0000-000000000001/meta.compacted.json</Key>
								<LastModified>2024-03-01T00:00:00.000Z</LastModified>
								<ETag>&quot;d42a22ddd183f61924c661b1c026c1ef&quot;</ETag>
								<Size>398</Size>
								<StorageClass>STANDARD</StorageClass>
							</Contents>
						</ListBucketResult>`))
						return
					}
				}
			},
		},
		{
			name:              "without prefix",
			prefix:            "",
			liveBlockIDs:      []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000000")},
			compactedBlockIDs: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000001")},
			tenant:            "single-tenant",
			httpHandler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if r.Method == getMethod {
						assert.Equal(t, "single-tenant/", r.URL.Query().Get("prefix"))

						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<ListBucketResult>
							<Name>blerg</Name>
							<Prefix></Prefix>
							<ContinuationToken></ContinuationToken>
							<KeyCount>2</KeyCount>
							<MaxKeys>100</MaxKeys>
							<EncodingType>url</EncodingType>
							<IsTruncated>false</IsTruncated>
							<Contents>
								<Key>single-tenant/00000000-0000-0000-0000-000000000000/meta.json</Key>
								<LastModified>2024-03-01T00:00:00.000Z</LastModified>
								<ETag>&quot;d42a22ddd183f61924c661b1c026c1ef&quot;</ETag>
								<Size>398</Size>
								<StorageClass>STANDARD</StorageClass>
							</Contents>
							
							<Contents>
								<Key>single-tenant/00000000-0000-0000-0000-000000000001/meta.compacted.json</Key>
								<LastModified>2024-03-01T00:00:00.000Z</LastModified>
								<ETag>&quot;d42a22ddd183f61924c661b1c026c1ef&quot;</ETag>
								<Size>398</Size>
								<StorageClass>STANDARD</StorageClass>
							</Contents>
						</ListBucketResult>`))
						return
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testServer(t, tc.httpHandler(t))
			r, _, _, err := NewNoConfirm(&Config{
				Region:                "blerg",
				AccessKey:             "test",
				SecretKey:             flagext.SecretWithValue("test"),
				Bucket:                "blerg",
				Prefix:                tc.prefix,
				Insecure:              true,
				Endpoint:              server.URL[7:],
				ListBlocksConcurrency: 1,
			})
			require.NoError(t, err)

			ctx := context.Background()
			blockIDs, compactedBlockIDs, noCompactBlockIDs, err := r.ListBlocks(ctx, tc.tenant)
			assert.NoError(t, err)

			assert.ElementsMatchf(t, tc.liveBlockIDs, blockIDs, "Block IDs did not match")
			assert.ElementsMatchf(t, tc.compactedBlockIDs, compactedBlockIDs, "Compacted block IDs did not match")
			assert.ElementsMatchf(t, tc.noCompactBlockIDs, noCompactBlockIDs, "Nocompact block IDs did not match")
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
			var httpHeader http.Header

			server := fakeServerWithHeader(t, &httpHeader)
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
			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, nil)
			require.Equal(t, tc.StorageClass, httpHeader.Get(storageClassHeader))
		})
	}
}

func TestDeleteVersioned_DoesNotDoublePrefix(t *testing.T) {
	const (
		prefix             = "a/b/c"
		name               = "overrides.json"
		etag               = `"etag123"`
		body               = `{}`
		expectedDeletePath = "/blerg/a/b/c/overrides/tenant-1/" + name
	)

	var capturedDeletePath string
	server := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			// New() probes the bucket with a list; ReadVersioned fetches the object body.
			if strings.HasSuffix(r.URL.Path, "/"+name) {
				w.Header().Set("ETag", etag)
				w.Header().Set("Last-Modified", "Fri, 01 Mar 2024 00:00:00 GMT")
				_, _ = w.Write([]byte(body))
				return
			}
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult></ListBucketResult>`))
		case http.MethodDelete:
			capturedDeletePath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})

	rw, err := NewVersionedReaderWriter(&Config{
		Region:    "blerg",
		AccessKey: "test",
		SecretKey: flagext.SecretWithValue("test"),
		Bucket:    "blerg",
		Prefix:    prefix,
		Insecure:  true,
		Endpoint:  server.URL[7:],
	})
	require.NoError(t, err)

	// minio strips the surrounding quotes from the ETag in the response.
	require.NoError(t, rw.DeleteVersioned(context.Background(), name, backend.KeyPath{"overrides", "tenant-1"}, backend.Version("etag123")))
	assert.Equal(t, expectedDeletePath, capturedDeletePath,
		"DELETE wire path must contain the configured prefix exactly once")
}

func testServer(t *testing.T, httpHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	assert.NotNil(t, httpHandler)
	server := httptest.NewServer(httpHandler)
	t.Cleanup(server.Close)
	return server
}

// directoryBucketName follows the naming scheme of S3 directory buckets, here one in a Local Zone.
const directoryBucketName = "tempo--euc1-ist1-az1--x-s3"

func TestNewConfirmsBucketWithListObjectsV2(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		expectedPrefix string
	}{
		{name: "without prefix", prefix: "", expectedPrefix: ""},
		{name: "with prefix", prefix: "tempo", expectedPrefix: "tempo/"},
		{name: "with trailing slash prefix", prefix: "a/b/c/", expectedPrefix: "a/b/c/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, true)
			_, _, _, err := New(&Config{
				Region:    "blerg",
				AccessKey: "test",
				SecretKey: flagext.SecretWithValue("test"),
				Bucket:    directoryBucketName,
				Prefix:    tc.prefix,
				Insecure:  true,
				Endpoint:  endpoint,
			})
			require.NoError(t, err)

			queries := bucket.listQueries()
			require.Len(t, queries, 1)
			assert.Equal(t, tc.expectedPrefix, queries[0].Get("prefix"))
		})
	}
}

func TestList(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111").String()
	bucket, endpoint := newFakeBucket(t, true,
		"tempo/tempo_cluster_seed.json",
		"tempo/tenant-1/index.json.gz",
		"tempo/tenant-2/"+blockID+"/meta.json",
		"tempo/tenant-3/"+blockID+"/data.parquet",
	)
	r, _, _, err := NewNoConfirm(&Config{
		Region:    "blerg",
		AccessKey: "test",
		SecretKey: flagext.SecretWithValue("test"),
		Bucket:    directoryBucketName,
		Prefix:    "tempo",
		Insecure:  true,
		Endpoint:  endpoint,
	})
	require.NoError(t, err)

	tenants, err := r.List(context.Background(), nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"tenant-1", "tenant-2", "tenant-3"}, tenants)
	assert.Greater(t, len(bucket.listQueries()), 1, "expected the listing to span several pages")
}

func TestListBlocks(t *testing.T) {
	var (
		liveBlockIDs = []uuid.UUID{
			uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			uuid.MustParse("77777777-7777-7777-7777-777777777777"),
			uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		}
		compactedBlockIDs = []uuid.UUID{
			uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		}
		noCompactBlockIDs = []uuid.UUID{
			uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		}
		keys = []string{"tempo/tenant/index.json.gz"}
	)
	for _, id := range liveBlockIDs {
		keys = append(keys, "tempo/tenant/"+id.String()+"/meta.json", "tempo/tenant/"+id.String()+"/data.parquet")
	}
	for _, id := range compactedBlockIDs {
		keys = append(keys, "tempo/tenant/"+id.String()+"/meta.compacted.json", "tempo/tenant/"+id.String()+"/data.parquet")
	}
	for _, id := range noCompactBlockIDs {
		keys = append(keys, "tempo/tenant/"+id.String()+"/nocompact.flg")
	}

	tests := []struct {
		name               string
		bucket             string
		directory          bool
		expectedStartAfter int
	}{
		{
			// each of the three shards lists from its lowest block ID
			name:               "general purpose bucket",
			bucket:             "blerg",
			expectedStartAfter: 3,
		},
		{
			name:               "directory bucket",
			bucket:             directoryBucketName,
			directory:          true,
			expectedStartAfter: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, tc.directory, keys...)
			r, _, _, err := NewNoConfirm(&Config{
				Region:                "blerg",
				AccessKey:             "test",
				SecretKey:             flagext.SecretWithValue("test"),
				Bucket:                tc.bucket,
				Prefix:                "tempo",
				Insecure:              true,
				Endpoint:              endpoint,
				ListBlocksConcurrency: 3,
			})
			require.NoError(t, err)

			blockIDs, compacted, noCompact, err := r.ListBlocks(context.Background(), "tenant")
			require.NoError(t, err)
			assert.ElementsMatch(t, liveBlockIDs, blockIDs)
			assert.ElementsMatch(t, compactedBlockIDs, compacted)
			assert.ElementsMatch(t, noCompactBlockIDs, noCompact)

			startAfter := map[string]struct{}{}
			for _, q := range bucket.listQueries() {
				if q.Has("start-after") {
					startAfter[q.Get("start-after")] = struct{}{}
				}
			}
			assert.Len(t, startAfter, tc.expectedStartAfter)
		})
	}
}

func TestIsDirectoryBucket(t *testing.T) {
	tests := []struct {
		bucket   string
		expected bool
	}{
		{bucket: "blerg", expected: false},
		{bucket: "tempo-x-s3", expected: false},
		{bucket: "amzn-s3-demo-bucket--usw2-az1--x-s3", expected: true},
		{bucket: directoryBucketName, expected: true},
		{bucket: "my-access-point--usw2-az1--xa-s3", expected: true},
	}

	for _, tc := range tests {
		t.Run(tc.bucket, func(t *testing.T) {
			assert.Equal(t, tc.expected, isDirectoryBucket(tc.bucket))
		})
	}
}

// fakeBucket is an in-memory bucket serving the list and delete calls of the backend. It only
// supports ListObjectsV2 and returns at most two entries per page. With directory set it also
// follows the listing rules of S3 directory buckets: start-after isn't supported, prefixes must
// end in "/" and keys aren't listed in lexicographical order.
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-differences.html
type fakeBucket struct {
	directory       bool
	sessionLifetime time.Duration

	mtx      sync.Mutex
	keys     map[string]struct{}
	queries  []url.Values
	sessions int
}

const (
	fakeSessionAccessKey = "session-access-key"
	fakeSessionToken     = "session-token"
)

type fakeListResult struct {
	XMLName               xml.Name           `xml:"ListBucketResult"`
	IsTruncated           bool               `xml:"IsTruncated"`
	NextContinuationToken string             `xml:"NextContinuationToken,omitempty"`
	Contents              []fakeListContents `xml:"Contents"`
	CommonPrefixes        []fakeListPrefix   `xml:"CommonPrefixes"`
}

type fakeListContents struct {
	Key string `xml:"Key"`
}

type fakeListPrefix struct {
	Prefix string `xml:"Prefix"`
}

func newFakeBucket(t *testing.T, directory bool, keys ...string) (*fakeBucket, string) {
	t.Helper()

	b := &fakeBucket{directory: directory, sessionLifetime: 5 * time.Minute, keys: map[string]struct{}{}}
	for _, key := range keys {
		b.keys[key] = struct{}{}
	}
	server := testServer(t, b.serveHTTP)
	return b, server.URL[7:] // [7:] -> strip http://
}

func (b *fakeBucket) listQueries() []url.Values {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return slices.Clone(b.queries)
}

func (b *fakeBucket) remainingKeys() []string {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return slices.Collect(maps.Keys(b.keys))
}

func (b *fakeBucket) createdSessions() int {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.sessions
}

func (b *fakeBucket) serveHTTP(w http.ResponseWriter, r *http.Request) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// path-style requests: /<bucket>/<key>
	_, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if r.Method == http.MethodGet && key == "" && r.URL.Query().Has("session") {
		b.createSession(w, r)
		return
	}
	if err := b.authorize(r); err != nil {
		w.WriteHeader(http.StatusForbidden)
		_ = xml.NewEncoder(w).Encode(minio.ErrorResponse{Code: "AccessDenied", Message: err.Error()})
		return
	}

	switch {
	case r.Method == http.MethodGet && key == "":
		b.list(w, r.URL.Query())
	case r.Method == http.MethodPut && r.Header.Get("X-Amz-Copy-Source") != "":
		source, _ := url.PathUnescape(r.Header.Get("X-Amz-Copy-Source"))
		_, sourceKey, _ := strings.Cut(strings.TrimPrefix(source, "/"), "/")
		if _, ok := b.keys[sourceKey]; !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = xml.NewEncoder(w).Encode(minio.ErrorResponse{Code: minio.NoSuchKey})
			return
		}
		b.keys[key] = struct{}{}
		_, _ = w.Write([]byte(`<CopyObjectResult><ETag>"etag"</ETag></CopyObjectResult>`))
	case r.Method == http.MethodDelete:
		delete(b.keys, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// signingScope returns the access key and service of the SigV4 Authorization header.
func signingScope(r *http.Request) (accessKey, service string) {
	// AWS4-HMAC-SHA256 Credential=<access key>/<date>/<region>/<service>/aws4_request, ...
	credential, _, _ := strings.Cut(strings.TrimPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential="), ",")
	parts := strings.Split(credential, "/")
	if len(parts) != 5 {
		return "", ""
	}
	return parts[0], parts[3]
}

// createSession serves CreateSession, which directory buckets authorize with IAM credentials.
func (b *fakeBucket) createSession(w http.ResponseWriter, r *http.Request) {
	if accessKey, service := signingScope(r); accessKey != "test" || service != "s3express" {
		w.WriteHeader(http.StatusForbidden)
		_ = xml.NewEncoder(w).Encode(minio.ErrorResponse{Code: "AccessDenied", Message: "CreateSession must be signed with IAM credentials for s3express"})
		return
	}
	b.sessions++

	var res createSessionResult
	res.Credentials.AccessKeyID = fakeSessionAccessKey
	res.Credentials.SecretAccessKey = "session-secret-key"
	res.Credentials.SessionToken = fakeSessionToken
	res.Credentials.Expiration = time.Now().Add(b.sessionLifetime)
	_ = xml.NewEncoder(w).Encode(res)
}

// authorize checks requests are signed like S3 expects: directory buckets take session
// credentials for the s3express service, except for CopyObject which takes IAM credentials.
func (b *fakeBucket) authorize(r *http.Request) error {
	accessKey, service := signingScope(r)
	switch {
	case !b.directory:
		if service != "s3" {
			return fmt.Errorf("expected a signature for s3, got %q", service)
		}
	case service != "s3express":
		return fmt.Errorf("expected a signature for s3express, got %q", service)
	case r.Header.Get("X-Amz-Copy-Source") != "":
		if accessKey != "test" || r.Header.Get("X-Amz-S3session-Token") != "" {
			return errors.New("CopyObject must be signed with IAM credentials")
		}
	case accessKey != fakeSessionAccessKey || r.Header.Get("X-Amz-S3session-Token") != fakeSessionToken:
		return errors.New("expected session credentials")
	}
	return nil
}

func (b *fakeBucket) list(w http.ResponseWriter, query url.Values) {
	b.queries = append(b.queries, query)

	prefix, delimiter, startAfter := query.Get("prefix"), query.Get("delimiter"), query.Get("start-after")
	switch {
	case query.Get("list-type") != "2":
		writeS3Error(w, "This bucket does not support ListObjects API. Consider using ListObjectsV2 API.")
		return
	case b.directory && query.Has("start-after"):
		writeS3Error(w, "start-after is not supported by directory buckets")
		return
	case b.directory && prefix != "" && !strings.HasSuffix(prefix, "/"):
		writeS3Error(w, "directory buckets only support prefixes that end in a delimiter")
		return
	}

	type entry struct {
		name     string
		isPrefix bool
	}
	var (
		entries  []entry
		prefixes = map[string]struct{}{}
	)
	for key := range b.keys {
		if !strings.HasPrefix(key, prefix) || (startAfter != "" && key <= startAfter) {
			continue
		}
		if i := strings.Index(key[len(prefix):], delimiter); delimiter != "" && i >= 0 {
			commonPrefix := key[:len(prefix)+i+len(delimiter)]
			if _, ok := prefixes[commonPrefix]; !ok {
				prefixes[commonPrefix] = struct{}{}
				entries = append(entries, entry{name: commonPrefix, isPrefix: true})
			}
			continue
		}
		entries = append(entries, entry{name: key})
	}
	slices.SortFunc(entries, func(a, c entry) int {
		if b.directory {
			return strings.Compare(c.name, a.name)
		}
		return strings.Compare(a.name, c.name)
	})

	// the continuation token is the offset of the next page
	pageSize := 2
	if maxKeys, _ := strconv.Atoi(query.Get("max-keys")); maxKeys > 0 && maxKeys < pageSize {
		pageSize = maxKeys
	}
	start, _ := strconv.Atoi(query.Get("continuation-token"))
	start = min(start, len(entries))
	end := min(start+pageSize, len(entries))

	res := fakeListResult{IsTruncated: end < len(entries)}
	if res.IsTruncated {
		res.NextContinuationToken = strconv.Itoa(end)
	}
	for _, e := range entries[start:end] {
		if e.isPrefix {
			res.CommonPrefixes = append(res.CommonPrefixes, fakeListPrefix{Prefix: e.name})
		} else {
			res.Contents = append(res.Contents, fakeListContents{Key: e.name})
		}
	}
	_ = xml.NewEncoder(w).Encode(res)
}

func writeS3Error(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = xml.NewEncoder(w).Encode(minio.ErrorResponse{Code: "InvalidRequest", Message: message})
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func metadataMockedHandler(t *testing.T) http.HandlerFunc {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/":
			err := r.ParseForm()
			require.NoError(t, err)

			if r.Form.Get("Action") != "AssumeRoleWithWebIdentity" {
				w.WriteHeader(400)
			}

			token, err := os.ReadFile(filepath.Join(cwd, "testdata/iam-token"))
			require.NoError(t, err)
			if r.Form.Get("WebIdentityToken") != string(token) {
				w.WriteHeader(400)
			}

			type xmlCreds struct {
				AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
				SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
				Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
				SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
			}

			assumeResponse := credentials.AssumeRoleWithWebIdentityResponse{
				Result: credentials.WebIdentityResult{
					Credentials: xmlCreds{
						AccessKey: defaultAccessKey,
						SecretKey: defaultSecretKey,
					},
				},
			}

			err1 := xml.NewEncoder(w).Encode(assumeResponse)
			require.NoError(t, err1)
		case "/latest/api/token":
			// Check for X-aws-ec2-metadata-token-ttl-seconds request header
			if r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") == "" {
				w.WriteHeader(400)
			}

			// Check X-aws-ec2-metadata-token-ttl-seconds is an integer
			secondsInt, err := strconv.Atoi(r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds"))
			if err != nil {
				w.WriteHeader(400)
			}

			// Generate a token, 40 character string, base64 encoded
			token := base64.StdEncoding.EncodeToString([]byte(RandStringBytesMaskImprSrc(40)))

			w.Header().Set("X-Aws-Ec2-Metadata-Token-Ttl-Seconds", strconv.Itoa(secondsInt))
			if _, err := w.Write([]byte(token)); err != nil {
				require.NoError(t, err)
			}
		case "/latest/meta-data/iam/security-credentials/":
			if _, err := w.Write([]byte("role-name\n")); err != nil {
				require.NoError(t, err)
			}
		case "/latest/meta-data/iam/security-credentials/role-name":
			creds := ec2RoleCredRespBody{
				LastUpdated:     timeNow(),
				Expiration:      timeNow().Add(1 * time.Hour),
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				Type:            "AWS-HMAC",
				Code:            "Success",
			}

			err := json.NewEncoder(w).Encode(creds)
			require.NoError(t, err)
		}
	}
}

func timeNow() time.Time { return time.Date(2024, 5, 12, 16, 21, 24, 42, time.UTC) }
