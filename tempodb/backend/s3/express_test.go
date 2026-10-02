package s3

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
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

			r, _, _, err := NewNoConfirm(&Config{
				Region:    "blerg",
				AccessKey: "test",
				SecretKey: flagext.SecretWithValue("test"),
				Bucket:    directoryBucketName,
				Insecure:  true,
				Endpoint:  endpoint,
			})
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

func TestMarkBlockCompactedDirectoryBucket(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	meta := "tenant/" + blockID.String() + "/meta.json"
	compactedMeta := "tenant/" + blockID.String() + "/meta.compacted.json"

	bucket, endpoint := newFakeBucket(t, true, meta)
	_, _, c, err := NewNoConfirm(&Config{
		Region:    "blerg",
		AccessKey: "test",
		SecretKey: flagext.SecretWithValue("test"),
		Bucket:    directoryBucketName,
		Insecure:  true,
		Endpoint:  endpoint,
	})
	require.NoError(t, err)

	// the fake bucket only accepts the copy when it's signed with IAM credentials
	require.NoError(t, c.MarkBlockCompacted(blockID, "tenant"))
	assert.ElementsMatch(t, []string{compactedMeta}, bucket.remainingKeys())
}

func TestS3ExpressTransportAnswersCreateSession(t *testing.T) {
	bucket, endpoint := newFakeBucket(t, true)
	creds := credentials.NewStaticV4("test", "test", "")
	transport := newS3ExpressTransport(http.DefaultTransport, creds, "blerg", "http://"+endpoint+"/"+directoryBucketName+"/?session")

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

func TestS3ExpressTransportAnswersMinioCreateSession(t *testing.T) {
	// minio-go sends its own CreateSession requests for directory buckets in the Availability Zones
	// it knows, and only decodes answers in the S3 namespace
	const (
		zonalEndpoint = "s3express-use1-az4.us-east-1.amazonaws.com"
		azBucket      = "tempo--use1-az4--x-s3"
	)
	bucket, endpoint := newFakeBucket(t, true)
	creds := credentials.NewStaticV4("test", "test", "")
	transport := newS3ExpressTransport(redirectTransport{host: endpoint}, creds, "us-east-1", "https://"+zonalEndpoint+"/"+azBucket+"/?session")

	core, err := minio.NewCore(zonalEndpoint, &minio.Options{
		Creds:        creds,
		Region:       "us-east-1",
		Secure:       true,
		Transport:    transport,
		BucketLookup: minio.BucketLookupPath,
	})
	require.NoError(t, err)

	_, err = core.ListObjectsV2(azBucket, "", "", "", "/", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, bucket.createdSessions())
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

func TestS3ExpressTransportCreateSessionError(t *testing.T) {
	_, endpoint := newFakeBucket(t, true)
	// the fake bucket rejects CreateSession signed with these credentials
	creds := credentials.NewStaticV4("other", "other", "")
	transport := newS3ExpressTransport(http.DefaultTransport, creds, "blerg", "http://"+endpoint+"/"+directoryBucketName+"/?session")

	req, err := http.NewRequest(http.MethodGet, "http://"+endpoint+"/"+directoryBucketName+"/?list-type=2", nil)
	require.NoError(t, err)
	_, err = transport.RoundTrip(req) //nolint:bodyclose // an error means no response
	require.ErrorContains(t, err, "error from CreateSession")
	require.ErrorContains(t, err, "AccessDenied")
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
	_, _, _, err := NewNoConfirm(&Config{
		AccessKey: "test",
		SecretKey: flagext.SecretWithValue("test"),
		Bucket:    directoryBucketName,
	})
	require.ErrorContains(t, err, "region is required")
}
