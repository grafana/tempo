package s3

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/grafana/dskit/flagext"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

const (
	getMethod          = "GET"
	putMethod          = "PUT"
	tagHeader          = "X-Amz-Tagging"
	storageClassHeader = "X-Amz-Storage-Class"

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

func TestCredentials(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	tests := []struct {
		name     string
		access   string
		secret   string
		envs     map[string]string
		profile  string
		expected credentials.Value
		irsa     bool
		imds     bool
		mocked   bool
	}{
		{
			name: "no-creds",
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
			irsa:   true,
			mocked: true,
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
			},
			imds:   true,
			mocked: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mocked == true {
				metadataSrv := httptest.NewServer(metadataMockedHandler(t))
				defer metadataSrv.Close()

				if tc.envs == nil {
					tc.envs = map[string]string{}
				}

				tc.envs["TEST_IAM_ENDPOINT"] = metadataSrv.URL
			}

			closer := envSetter(tc.envs)
			defer t.Cleanup(closer)

			c := &Config{}
			if tc.access != "" {
				c.AccessKey = tc.access
				c.SecretKey = flagext.SecretWithValue(tc.secret)
			}

			creds, err := fetchCreds(c)
			assert.NoError(t, err)

			realCreds, err := creds.Get()
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
			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, nil)

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
			_ = w.Write(ctx, "object", backend.KeyPath{"test"}, bytes.NewReader([]byte{}), 0, nil)
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

func envSetter(envs map[string]string) (closer func()) {
	originalEnvs := map[string]string{}

	for name, value := range envs {
		if originalValue, ok := os.LookupEnv(name); ok {
			originalEnvs[name] = originalValue
		}
		_ = os.Setenv(name, value)
	}

	return func() {
		for name := range envs {
			origValue, has := originalEnvs[name]
			if has {
				_ = os.Setenv(name, origValue)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	}
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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/" {
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
		} else if r.URL.String() == "/latest/api/token" {
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
		} else if r.URL.String() == "/latest/meta-data/iam/security-credentials/" {
			if _, err := w.Write([]byte("role-name\n")); err != nil {
				require.NoError(t, err)
			}
		} else if r.URL.String() == "/latest/meta-data/iam/security-credentials/role-name" {
			creds := ec2RoleCredRespBody{
				LastUpdated:     time.Now(),
				Expiration:      time.Now().Add(1 * time.Hour),
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				Type:            "AWS-HMAC",
				Code:            "Success",
			}

			err := json.NewEncoder(w).Encode(creds)
			require.NoError(t, err)
		}
	})
}
