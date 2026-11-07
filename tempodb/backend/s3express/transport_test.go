package s3express

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/signer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessions(t *testing.T) {
	tests := []struct {
		name             string
		sessionLifetime  time.Duration
		expectedSessions int
	}{
		{
			name:             "session is reused",
			sessionLifetime:  5 * time.Minute,
			expectedSessions: 1,
		},
		{
			// sessions that expire within sessionLeeway are replaced before every request
			name:             "expiring session is replaced",
			sessionLifetime:  30 * time.Second,
			expectedSessions: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, server := newFakeBucket(t, "tenant-1/index.json.gz")
			bucket.update(func(b *fakeBucket) { b.sessionLifetime = tc.sessionLifetime })

			r, _, _, err := NewNoConfirm(fakeConfig(server))
			require.NoError(t, err)

			for range 3 {
				tenants, err := r.List(context.Background(), nil)
				require.NoError(t, err)
				assert.Equal(t, []string{"tenant-1"}, tenants)
			}
			assert.Equal(t, tc.expectedSessions, bucket.createdSessions())
			assert.Empty(t, bucket.rejectedRequests())
		})
	}
}

func TestTransportSharesSessions(t *testing.T) {
	// the clock of the transport and the fake bucket, moved forward to replace the session
	var elapsed atomic.Int64
	now := func() time.Time { return time.Now().Add(time.Duration(elapsed.Load())) }

	bucket, server := newFakeBucket(t)
	bucket.update(func(b *fakeBucket) {
		b.now = now
		// keeps the first CreateSession in flight while the other requests arrive
		b.sessionDelay = 100 * time.Millisecond
	})
	transport := newTransport(server.Client().Transport, iamCredentials(), "eu-central-1", fakeBucketURL(server))
	transport.now = now

	listConcurrently := func() {
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				req, err := http.NewRequest(http.MethodGet, fakeBucketURL(server).String()+"?list-type=2", nil)
				if !assert.NoError(t, err) {
					return
				}
				resp, err := transport.RoundTrip(req)
				if !assert.NoError(t, err) {
					return
				}
				defer resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
		wg.Wait()
	}

	// the requests wait for the session the first of them creates
	listConcurrently()
	assert.Equal(t, 1, bucket.createdSessions())

	// and for the session that replaces it once it's about to expire
	elapsed.Store(int64(4*time.Minute + 30*time.Second))
	listConcurrently()
	assert.Equal(t, 2, bucket.createdSessions())
	assert.Empty(t, bucket.rejectedRequests())
}

func TestTransportAnswersCreateSession(t *testing.T) {
	bucket, server := newFakeBucket(t)
	transport := newTransport(server.Client().Transport, iamCredentials(), "eu-central-1", fakeBucketURL(server))

	// minio-go's own CreateSession request, possibly to the wrong host, is answered with the
	// response that created the transport's session
	req, err := http.NewRequest(http.MethodGet, "https://invalid.example.com/?session", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var result createSessionResult
	require.NoError(t, xml.Unmarshal(body, &result))
	assert.Equal(t, "session-access-key-1", result.Credentials.AccessKeyID)
	assert.Equal(t, "session-token-1", result.Credentials.SessionToken)

	// which signs the requests
	req, err = http.NewRequest(http.MethodGet, fakeBucketURL(server).String()+"?list-type=2", nil)
	require.NoError(t, err)
	resp, err = transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, bucket.createdSessions())
	assert.Empty(t, bucket.rejectedRequests())
}

func TestTransportPassesAnonymousRequests(t *testing.T) {
	var received *http.Request
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		received = req
		return response(http.StatusOK, ""), nil
	})
	transport := newTransport(next, credentials.NewStatic("", "", "", credentials.SignatureAnonymous), "eu-central-1", testBucketURL())

	// sent as they are: unsigned, to the host they're addressed to, without a session
	req, err := http.NewRequest(http.MethodGet, "https://other.example.com/object", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Same(t, req, received)
	assert.Empty(t, received.Header.Get("Authorization"))
}

func TestTransportErrors(t *testing.T) {
	tests := []struct {
		name          string
		creds         *credentials.Credentials
		createSession func() (*http.Response, error)
		expectedErr   string
	}{
		{
			name:        "no credentials",
			creds:       credentials.New(failingProvider{}),
			expectedErr: "failed to get credentials: no credentials",
		},
		{
			name:          "CreateSession isn't sent",
			createSession: func() (*http.Response, error) { return nil, errors.New("connection refused") },
			expectedErr:   "error from CreateSession: connection refused",
		},
		{
			name: "CreateSession response can't be read",
			createSession: func() (*http.Response, error) {
				resp := response(http.StatusOK, "")
				resp.Body = io.NopCloser(failingReader{})
				return resp, nil
			},
			expectedErr: "error reading CreateSession response: connection reset",
		},
		{
			name: "CreateSession is denied",
			createSession: func() (*http.Response, error) {
				return response(http.StatusForbidden, "<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>"), nil
			},
			expectedErr: "AccessDenied",
		},
		{
			name: "CreateSession fails without an S3 error",
			createSession: func() (*http.Response, error) {
				return response(http.StatusServiceUnavailable, "<html>unavailable</html>"), nil
			},
			expectedErr: "503 Service Unavailable",
		},
		{
			name:          "CreateSession response isn't XML",
			createSession: func() (*http.Response, error) { return response(http.StatusOK, "not xml"), nil },
			expectedErr:   "error decoding CreateSession response",
		},
		{
			name: "CreateSession response has no credentials",
			createSession: func() (*http.Response, error) {
				return response(http.StatusOK, `<CreateSessionResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Credentials></Credentials></CreateSessionResult>`), nil
			},
			expectedErr: "no session credentials",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var sent atomic.Int32
			next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Has("session") {
					return tc.createSession()
				}
				sent.Add(1)
				return response(http.StatusOK, ""), nil
			})
			creds := tc.creds
			if creds == nil {
				creds = iamCredentials()
			}
			transport := newTransport(next, creds, "eu-central-1", testBucketURL())

			body := &closeRecorder{Reader: strings.NewReader("data")}
			req, err := http.NewRequest(http.MethodPut, testBucketURL().String()+"object", body)
			require.NoError(t, err)
			_, err = transport.RoundTrip(req) //nolint:bodyclose // an error means no response
			require.ErrorContains(t, err, tc.expectedErr)
			assert.True(t, body.closed.Load(), "expected the request body to be closed")

			// minio-go's own CreateSession requests fail the same way
			req, err = http.NewRequest(http.MethodGet, testBucketURL().String()+"?session", nil)
			require.NoError(t, err)
			_, err = transport.RoundTrip(req) //nolint:bodyclose // an error means no response
			require.ErrorContains(t, err, tc.expectedErr)

			assert.Zero(t, sent.Load(), "expected no request to reach the bucket")
		})
	}
}

func TestTransportDropsRejectedSessions(t *testing.T) {
	tests := []struct {
		status           int
		expectedSessions int
	}{
		// the session may have expired or been revoked
		{status: http.StatusBadRequest, expectedSessions: 2},
		{status: http.StatusForbidden, expectedSessions: 2},
		// errors about the object or the bucket keep it
		{status: http.StatusNotFound, expectedSessions: 1},
		{status: http.StatusPreconditionFailed, expectedSessions: 1},
		{status: http.StatusInternalServerError, expectedSessions: 1},
	}

	for _, tc := range tests {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			var sessions int
			status := tc.status
			next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Has("session") {
					sessions++
					return response(http.StatusOK, sessionResponseBody(t, sessions)), nil
				}
				return response(status, ""), nil
			})
			transport := newTransport(next, iamCredentials(), "eu-central-1", testBucketURL())

			roundTrip := func(expectedStatus int) {
				req, err := http.NewRequest(http.MethodGet, testBucketURL().String()+"object", nil)
				require.NoError(t, err)
				resp, err := transport.RoundTrip(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, expectedStatus, resp.StatusCode)
			}
			roundTrip(tc.status)
			status = http.StatusOK
			roundTrip(http.StatusOK)
			assert.Equal(t, tc.expectedSessions, sessions)
		})
	}

	t.Run("replaced session", func(t *testing.T) {
		transport := newTransport(http.DefaultTransport, iamCredentials(), "eu-central-1", testBucketURL())
		transport.session = session{Value: credentials.Value{SessionToken: "new"}}

		// a request signed with the session that was replaced since doesn't drop the new one
		transport.dropSession("old")
		assert.Equal(t, "new", transport.session.SessionToken)
		transport.dropSession("new")
		assert.Empty(t, transport.session.SessionToken)
	})
}

func TestVerifySignature(t *testing.T) {
	// the fake bucket checks the signature of every request, check it rejects wrong ones
	errs := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		errs <- verifySignature(r, "secret")
	}))
	t.Cleanup(server.Close)

	send := func(req *http.Request) error {
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return <-errs
	}
	signed := func(secret string, update func(req *http.Request)) *http.Request {
		req, err := http.NewRequest(http.MethodPut, server.URL+"/bucket/a/key?uploadId=1&partNumber=2", strings.NewReader("data"))
		require.NoError(t, err)
		req.Header.Set("X-Amz-Content-Sha256", "3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7") // SHA-256 of "data"
		req.Header.Set("X-Amz-S3session-Token", "token")
		update(req)
		return signer.SignV4Express(*req, "key", secret, "token", "eu-central-1")
	}

	require.NoError(t, send(signed("secret", func(*http.Request) {})))
	require.Error(t, send(signed("other", func(*http.Request) {})), "signed with another secret key")

	req := signed("secret", func(*http.Request) {})
	req.Header.Set("X-Amz-S3session-Token", "other")
	require.Error(t, send(req), "a signed header changed")

	req = signed("secret", func(*http.Request) {})
	req.URL.RawQuery = "uploadId=1&partNumber=3"
	require.Error(t, send(req), "the query changed")

	req = signed("secret", func(req *http.Request) { req.Body = io.NopCloser(strings.NewReader("diff")) })
	require.Error(t, send(req), "the body doesn't match its hash")
}

// testBucketURL is the URL of a directory bucket the tests don't send requests to.
func testBucketURL() *url.URL {
	return &url.URL{Scheme: "https", Host: directoryBucketName + ".s3express-euc1-ist1-az1.eu-central-1.amazonaws.com", Path: "/"}
}

func iamCredentials() *credentials.Credentials {
	return credentials.NewStaticV4(iamAccessKey, iamSecretKey, "")
}

// sessionResponseBody returns a CreateSession response for the nth session.
func sessionResponseBody(t *testing.T, n int) string {
	var result createSessionResult
	result.Credentials.AccessKeyID = fmt.Sprintf("session-access-key-%d", n)
	result.Credentials.SecretAccessKey = fmt.Sprintf("session-secret-key-%d", n)
	result.Credentials.SessionToken = fmt.Sprintf("session-token-%d", n)
	result.Credentials.Expiration = time.Now().Add(5 * time.Minute)
	body, err := xml.Marshal(result)
	require.NoError(t, err)
	return string(body)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// failingProvider is a credentials provider that fails.
type failingProvider struct{}

func (failingProvider) Retrieve() (credentials.Value, error) {
	return credentials.Value{}, errors.New("no credentials")
}

func (failingProvider) RetrieveWithCredContext(*credentials.CredContext) (credentials.Value, error) {
	return credentials.Value{}, errors.New("no credentials")
}

func (failingProvider) IsExpired() bool {
	return true
}

// failingReader is a body that fails to be read.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

// closeRecorder is a request body that records whether it was closed.
type closeRecorder struct {
	io.Reader
	closed atomic.Bool
}

func (c *closeRecorder) Close() error {
	c.closed.Store(true)
	return nil
}
