package s3

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3ExpressSessions(t *testing.T) {
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
			// sessions that expire within s3ExpressSessionLeeway are replaced before every request
			name:             "expiring session is replaced",
			sessionLifetime:  30 * time.Second,
			expectedSessions: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, true, "tenant-1/index.json.gz")
			bucket.sessionLifetime = tc.sessionLifetime

			r, _, _, err := NewNoConfirm(fakeConfig(directoryBucketName, endpoint))
			require.NoError(t, err)

			for range 3 {
				tenants, err := r.List(context.Background(), nil)
				require.NoError(t, err)
				assert.Equal(t, []string{"tenant-1"}, tenants)
			}
			assert.Equal(t, tc.expectedSessions, bucket.createdSessions())
		})
	}
}

func TestS3ExpressTransportSharesSessions(t *testing.T) {
	// the clock of the transport and the fake bucket, moved forward to replace the session
	var elapsed atomic.Int64
	now := func() time.Time { return time.Now().Add(time.Duration(elapsed.Load())) }

	bucket, endpoint := newFakeBucket(t, true)
	bucket.now = now
	// keeps the first CreateSession in flight while the other requests arrive
	bucket.sessionDelay = 100 * time.Millisecond
	transport := newS3ExpressTransport(http.DefaultTransport, credentials.NewStaticV4("test", "test", ""), "blerg", fakeBucketURL(endpoint))
	transport.now = now

	listConcurrently := func() {
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				req, err := http.NewRequest(http.MethodGet, fakeBucketURL(endpoint).String()+"?list-type=2", nil)
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
}

func TestS3ExpressTransportAnswersCreateSession(t *testing.T) {
	bucket, endpoint := newFakeBucket(t, true)
	transport := newS3ExpressTransport(http.DefaultTransport, credentials.NewStaticV4("test", "test", ""), "blerg", fakeBucketURL(endpoint))

	// minio-go's own CreateSession request, possibly to the wrong host, is answered locally
	req, err := http.NewRequest(http.MethodGet, "http://invalid.example.com/?session", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var result createSessionResult
	require.NoError(t, xml.Unmarshal(body, &result))
	assert.Equal(t, fakeSessionAccessKey, result.Credentials.AccessKeyID)
	assert.Equal(t, fakeSessionToken, result.Credentials.SessionToken)
	assert.Equal(t, 1, bucket.createdSessions())
}

func TestS3ExpressTransportCreateSessionError(t *testing.T) {
	_, endpoint := newFakeBucket(t, true)
	// the fake bucket rejects CreateSession signed with these credentials
	transport := newS3ExpressTransport(http.DefaultTransport, credentials.NewStaticV4("other", "other", ""), "blerg", fakeBucketURL(endpoint))

	req, err := http.NewRequest(http.MethodGet, fakeBucketURL(endpoint).String()+"?list-type=2", nil)
	require.NoError(t, err)
	_, err = transport.RoundTrip(req) //nolint:bodyclose // an error means no response
	require.ErrorContains(t, err, "error from CreateSession")
	require.ErrorContains(t, err, "AccessDenied")
}

func TestS3ExpressTransportRejectsStreamingSignatures(t *testing.T) {
	// over http, minio-go signs uploads with streaming signatures, which sign every chunk of the
	// body with minio-go's signature
	bucket, endpoint := newFakeBucket(t, true)
	transport := newS3ExpressTransport(http.DefaultTransport, credentials.NewStaticV4("test", "test", ""), "blerg", fakeBucketURL(endpoint))

	body := &closeRecorder{Reader: strings.NewReader("data")}
	req, err := http.NewRequest(http.MethodPut, fakeBucketURL(endpoint).String()+"object", body)
	require.NoError(t, err)
	req.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")

	_, err = transport.RoundTrip(req) //nolint:bodyclose // an error means no response
	require.ErrorContains(t, err, "S3 directory buckets require https")
	assert.True(t, body.closed.Load(), "expected the request body to be closed")
	assert.Empty(t, bucket.requestHosts(), "expected the request not to be sent")
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

func TestMarkBlockCompactedDirectoryBucket(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	meta := "tenant/" + blockID.String() + "/meta.json"
	compactedMeta := "tenant/" + blockID.String() + "/meta.compacted.json"

	bucket, endpoint := newFakeBucket(t, true, meta)
	_, _, c, err := NewNoConfirm(fakeConfig(directoryBucketName, endpoint))
	require.NoError(t, err)

	// the fake bucket only accepts the copy when it's signed with IAM credentials
	require.NoError(t, c.MarkBlockCompacted(blockID, "tenant"))
	assert.ElementsMatch(t, []string{compactedMeta}, bucket.remainingKeys())
}

func TestDirectoryBucketRequestHosts(t *testing.T) {
	tests := []struct {
		name           string
		bucket         string
		region         string
		forcePathStyle bool
		expectedHost   string
	}{
		{
			// minio-go sends the requests for directory buckets in the Availability Zones it knows to
			// the S3 Regional endpoint, and creates sessions for them itself, which the transport
			// answers
			name:         "Availability Zone",
			bucket:       "tempo--use1-az4--x-s3",
			region:       "us-east-1",
			expectedHost: "tempo--use1-az4--x-s3.s3express-use1-az4.us-east-1.amazonaws.com",
		},
		{
			name:         "Local Zone",
			bucket:       directoryBucketName,
			region:       "eu-central-1",
			expectedHost: directoryBucketName + ".s3express-euc1-ist1-az1.eu-central-1.amazonaws.com",
		},
		{
			name:           "path style",
			bucket:         "tempo--use1-az4--x-s3",
			region:         "us-east-1",
			forcePathStyle: true,
			expectedHost:   "s3express-use1-az4.us-east-1.amazonaws.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, true)
			cfg := &Config{Bucket: tc.bucket, Region: tc.region, ForcePathStyle: tc.forcePathStyle}
			core, err := newCore(cfg, credentials.NewStaticV4("test", "test", ""), redirectTransport{host: endpoint}, false)
			require.NoError(t, err)

			_, err = core.ListObjectsV2(tc.bucket, "", "", "", "/", 0)
			require.NoError(t, err)
			// CreateSession and the listing
			assert.Equal(t, []string{tc.expectedHost, tc.expectedHost}, bucket.requestHosts())
		})
	}
}

// redirectTransport sends every request to host over http, whatever host it was addressed to.
type redirectTransport struct {
	host string
}

func (r redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = r.host
	return http.DefaultTransport.RoundTrip(req)
}

func TestDirectoryBucketEndpoint(t *testing.T) {
	const zonalEndpoint = "s3express-euc1-ist1-az1.eu-central-1.amazonaws.com"

	tests := []struct {
		name             string
		endpoint         string
		insecure         bool
		lookup           minio.BucketLookupType
		expectedEndpoint string
		expectedLookup   minio.BucketLookupType
		expectedURL      string
	}{
		{
			name:             "Zonal endpoint by default",
			expectedEndpoint: zonalEndpoint,
			expectedLookup:   minio.BucketLookupDNS,
			expectedURL:      "https://" + directoryBucketName + "." + zonalEndpoint + "/",
		},
		{
			name:             "amazonaws.com endpoint with a port",
			endpoint:         zonalEndpoint + ":443",
			expectedEndpoint: zonalEndpoint + ":443",
			expectedLookup:   minio.BucketLookupDNS,
			expectedURL:      "https://" + directoryBucketName + "." + zonalEndpoint + ":443/",
		},
		{
			name:             "path style",
			lookup:           minio.BucketLookupPath,
			expectedEndpoint: zonalEndpoint,
			expectedLookup:   minio.BucketLookupPath,
			expectedURL:      "https://" + zonalEndpoint + "/" + directoryBucketName + "/",
		},
		{
			name:             "other endpoint",
			endpoint:         "localhost:9000",
			insecure:         true,
			expectedEndpoint: "localhost:9000",
			expectedLookup:   minio.BucketLookupPath,
			expectedURL:      "http://localhost:9000/" + directoryBucketName + "/",
		},
		{
			name:             "other endpoint with virtual-hosted style",
			endpoint:         "s3.example.com",
			lookup:           minio.BucketLookupDNS,
			expectedEndpoint: "s3.example.com",
			expectedLookup:   minio.BucketLookupDNS,
			expectedURL:      "https://" + directoryBucketName + ".s3.example.com/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, lookup, bucketURL, err := directoryBucketEndpoint(&Config{
				Bucket:   directoryBucketName,
				Region:   "eu-central-1",
				Endpoint: tc.endpoint,
				Insecure: tc.insecure,
			}, tc.lookup)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedEndpoint, endpoint)
			assert.Equal(t, tc.expectedLookup, lookup)
			assert.Equal(t, tc.expectedURL, bucketURL.String())
		})
	}
}

func TestS3ExpressZonalEndpoint(t *testing.T) {
	tests := []struct {
		bucket   string
		expected string
	}{
		{bucket: "neta-prod-tempo--euc1-ist1-az1--x-s3", expected: "s3express-euc1-ist1-az1.eu-central-1.amazonaws.com"},
		{bucket: "tempo--euc1-az1--x-s3", expected: "s3express-euc1-az1.eu-central-1.amazonaws.com"},
		{bucket: "blerg", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.bucket, func(t *testing.T) {
			assert.Equal(t, tc.expected, s3ExpressZonalEndpoint(tc.bucket, "eu-central-1"))
		})
	}
}

func TestDirectoryBucketRequiresRegion(t *testing.T) {
	cfg := fakeConfig(directoryBucketName, "")
	cfg.Region = ""
	_, _, _, err := NewNoConfirm(cfg)
	require.ErrorContains(t, err, "region is required")
}

// fakeBucketURL returns the URL of the fake directory bucket at endpoint, for path-style requests
// over http.
func fakeBucketURL(endpoint string) *url.URL {
	return &url.URL{Scheme: "http", Host: endpoint, Path: "/" + directoryBucketName + "/"}
}
