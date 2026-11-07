package s3express

import (
	"bytes"
	"context"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	minio "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/s3"
)

// TestReadWrite writes, reads, lists, compacts and clears a block in a directory bucket. It runs
// against a fake directory bucket, and against a real one when TEMPO_TEST_S3_DIRECTORY_BUCKET and
// TEMPO_TEST_S3_DIRECTORY_BUCKET_REGION are set, with the credentials of the environment, for
// example AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.
func TestReadWrite(t *testing.T) {
	t.Run("fake", func(t *testing.T) {
		bucket, server := newFakeBucket(t)
		cfg := fakeConfig(server)
		cfg.Prefix = "tempo/"
		testReadWrite(t, cfg)
		assert.Empty(t, bucket.rejectedRequests())
	})

	t.Run("aws", func(t *testing.T) {
		bucket, region := os.Getenv("TEMPO_TEST_S3_DIRECTORY_BUCKET"), os.Getenv("TEMPO_TEST_S3_DIRECTORY_BUCKET_REGION")
		if bucket == "" || region == "" {
			t.Skip("set TEMPO_TEST_S3_DIRECTORY_BUCKET and TEMPO_TEST_S3_DIRECTORY_BUCKET_REGION to run against an S3 directory bucket")
		}
		cfg := &Config{}
		cfg.Bucket = bucket
		cfg.Region = region
		// keeps the test away from any other data in the bucket
		cfg.Prefix = "tempo-test-" + uuid.NewString() + "/"
		testReadWrite(t, cfg)
	})
}

func testReadWrite(t *testing.T, cfg *Config) {
	ctx := context.Background()
	r, w, c, err := New(cfg)
	require.NoError(t, err)

	const tenant = "tenant"
	blockID := uuid.New()
	keypath := backend.KeyPathForBlock(blockID, tenant)
	t.Cleanup(func() {
		if err := c.ClearBlock(blockID, tenant); err != nil {
			t.Logf("failed to clear block %s: %v", blockID, err)
		}
	})

	meta := []byte(`{"format":"vParquet4"}`)
	require.NoError(t, w.Write(ctx, backend.MetaName, keypath, bytes.NewReader(meta), int64(len(meta)), nil))

	// a multipart upload, with a first part of the minimum size of parts that aren't the last
	parts := [][]byte{bytes.Repeat([]byte{'a'}, 5<<20), []byte("the last part")}
	var tracker backend.AppendTracker
	for _, part := range parts {
		tracker, err = w.Append(ctx, "data.parquet", keypath, tracker, part)
		require.NoError(t, err)
	}
	require.NoError(t, w.CloseAppend(ctx, tracker))

	rc, size, err := r.Read(ctx, backend.MetaName, keypath, nil)
	require.NoError(t, err)
	actual, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, meta, actual)
	assert.Equal(t, int64(len(meta)), size)

	lastPart := make([]byte, len(parts[1]))
	require.NoError(t, r.ReadRange(ctx, "data.parquet", keypath, uint64(len(parts[0])), lastPart, nil))
	assert.Equal(t, parts[1], lastPart)

	tenants, err := r.List(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{tenant}, tenants)

	var found []string
	require.NoError(t, r.Find(ctx, backend.KeyPath{tenant}, func(m backend.FindMatch) {
		found = append(found, m.Key)
	}))
	assert.ElementsMatch(t, []string{
		tenant + "/" + blockID.String() + "/" + backend.MetaName,
		tenant + "/" + blockID.String() + "/data.parquet",
	}, found)

	blockIDs, compactedBlockIDs, _, err := r.ListBlocks(ctx, tenant)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{blockID}, blockIDs)
	assert.Empty(t, compactedBlockIDs)

	require.NoError(t, c.MarkBlockCompacted(blockID, tenant))
	blockIDs, compactedBlockIDs, _, err = r.ListBlocks(ctx, tenant)
	require.NoError(t, err)
	assert.Empty(t, blockIDs)
	assert.Equal(t, []uuid.UUID{blockID}, compactedBlockIDs)

	require.NoError(t, c.ClearBlock(blockID, tenant))
	_, _, err = r.Read(ctx, "data.parquet", keypath, nil)
	require.ErrorIs(t, err, backend.ErrDoesNotExist)
	blockIDs, compactedBlockIDs, _, err = r.ListBlocks(ctx, tenant)
	require.NoError(t, err)
	assert.Empty(t, blockIDs)
	assert.Empty(t, compactedBlockIDs)
}

func TestNewConfirmsBucket(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		expectedPrefix string
	}{
		{name: "without prefix", expectedPrefix: ""},
		{name: "with prefix", prefix: "tempo", expectedPrefix: "tempo/"},
		{name: "with trailing slash prefix", prefix: "a/b/c/", expectedPrefix: "a/b/c/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, server := newFakeBucket(t)
			cfg := fakeConfig(server)
			cfg.Prefix = tc.prefix
			_, _, _, err := New(cfg)
			require.NoError(t, err)

			queries := bucket.listQueries()
			require.Len(t, queries, 1)
			assert.Equal(t, "2", queries[0].Get("list-type"))
			assert.Equal(t, tc.expectedPrefix, queries[0].Get("prefix"))
		})
	}
}

func TestNewValidatesConfig(t *testing.T) {
	bucket, server := newFakeBucket(t)
	cfg := fakeConfig(server)
	cfg.Insecure = true

	_, _, _, err := New(cfg)
	require.ErrorContains(t, err, "insecure isn't supported")
	_, _, _, err = NewNoConfirm(cfg)
	require.ErrorContains(t, err, "insecure isn't supported")
	_, err = NewVersionedReaderWriter(cfg)
	require.ErrorContains(t, err, "insecure isn't supported")
	assert.Empty(t, bucket.requestHosts(), "expected no request to the bucket")

	_, _, _, err = New(nil)
	require.ErrorContains(t, err, "config is nil")
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name             string
		update           func(cfg *Config)
		expectedErr      string
		expectedWarnings int
	}{
		{
			name:   "valid",
			update: func(*Config) {},
		},
		{
			name:        "general purpose bucket",
			update:      func(cfg *Config) { cfg.Bucket = "tempo" },
			expectedErr: `bucket "tempo" isn't an S3 directory bucket`,
		},
		{
			name:        "no region",
			update:      func(cfg *Config) { cfg.Region = "" },
			expectedErr: "region is required",
		},
		{
			name:        "no zone ID and no endpoint",
			update:      func(cfg *Config) { cfg.Bucket = "tempo--x-s3" },
			expectedErr: "no zone ID",
		},
		{
			name: "no zone ID with an endpoint",
			update: func(cfg *Config) {
				cfg.Bucket = "tempo--x-s3"
				cfg.Endpoint = "s3.example.com"
			},
		},
		{
			name:        "insecure",
			update:      func(cfg *Config) { cfg.Insecure = true },
			expectedErr: "insecure isn't supported",
		},
		{
			name:             "list_blocks_concurrency",
			update:           func(cfg *Config) { cfg.ListBlocksConcurrency = 3 },
			expectedWarnings: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			cfg.Bucket = directoryBucketName
			cfg.Region = "eu-central-1"
			tc.update(cfg)

			warnings, err := cfg.Validate()
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, warnings, tc.expectedWarnings)
		})
	}
}

func TestRegisterFlagsAndApplyDefaults(t *testing.T) {
	// the s3 backend registers the flags of the options both backends take
	f := flag.NewFlagSet("", flag.PanicOnError)
	(&s3.Config{}).RegisterFlagsAndApplyDefaults("storage.trace", f)

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("storage.trace", f)

	assert.Equal(t, 1, cfg.ListBlocksConcurrency)
	// defaults of the s3 backend
	assert.Equal(t, 2, cfg.HedgeRequestsUpTo)
	assert.Equal(t, "VersionTLS12", cfg.MinVersion)
}

func TestList(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111").String()
	bucket, server := newFakeBucket(t,
		"tempo/tempo_cluster_seed.json",
		"tempo/tenant-1/index.json.gz",
		"tempo/tenant-2/"+blockID+"/meta.json",
		"tempo/tenant-3/"+blockID+"/data.parquet",
	)
	cfg := fakeConfig(server)
	cfg.Prefix = "tempo"
	r, _, _, err := NewNoConfirm(cfg)
	require.NoError(t, err)

	tenants, err := r.List(context.Background(), nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"tenant-1", "tenant-2", "tenant-3"}, tenants)
	assert.Greater(t, len(bucket.listQueries()), 1, "expected the listing to span several pages")

	blocks, err := r.List(context.Background(), backend.KeyPath{"tenant-2"})
	require.NoError(t, err)
	assert.Equal(t, []string{blockID}, blocks)
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
		keys = []string{"tempo/tenant/index.json.gz", "tempo/tenant/not-a-block/meta.json"}
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

	bucket, server := newFakeBucket(t, keys...)
	cfg := fakeConfig(server)
	cfg.Prefix = "tempo"
	r, _, _, err := NewNoConfirm(cfg)
	require.NoError(t, err)

	blockIDs, compacted, noCompact, err := r.ListBlocks(context.Background(), "tenant")
	require.NoError(t, err)
	assert.ElementsMatch(t, liveBlockIDs, blockIDs)
	assert.ElementsMatch(t, compactedBlockIDs, compacted)
	assert.ElementsMatch(t, noCompactBlockIDs, noCompact)

	// one listing, over several pages, which the fake bucket rejects with start-after
	queries := bucket.listQueries()
	assert.Greater(t, len(queries), 1, "expected the listing to span several pages")
	for _, q := range queries {
		assert.Equal(t, "tempo/tenant/", q.Get("prefix"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, err = r.ListBlocks(ctx, "tenant")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFind(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{name: "without prefix"},
		{name: "with prefix", prefix: "a/b/c/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keys := []string{"tenant/index.json.gz", "tenant/block-1/meta.json", "tenant/block-2/meta.compacted.json"}
			var objects []string
			for _, key := range append(slices.Clone(keys), "other/index.json.gz") {
				objects = append(objects, tc.prefix+key)
			}
			bucket, server := newFakeBucket(t, objects...)
			cfg := fakeConfig(server)
			cfg.Prefix = tc.prefix
			r, _, _, err := NewNoConfirm(cfg)
			require.NoError(t, err)

			var found []string
			err = r.Find(context.Background(), backend.KeyPath{"tenant"}, func(m backend.FindMatch) {
				found = append(found, m.Key)
				assert.True(t, m.Modified.Equal(fakeModTime), "unexpected modification time %s", m.Modified)
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, keys, found)
			assert.Greater(t, len(bucket.listQueries()), 1, "expected the listing to span several pages")
		})
	}
}

func TestClearBlock(t *testing.T) {
	var (
		blockID      = uuid.MustParse("11111111-1111-1111-1111-111111111111")
		otherBlockID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
		keys         []string
		otherKeys    = []string{
			"tempo/tenant/" + otherBlockID.String() + "/meta.json",
			"tempo/tenant/" + otherBlockID.String() + "/data.parquet",
		}
	)
	// more objects than fit in one page of the fake bucket
	for _, name := range []string{"meta.json", "data.parquet", "bloom-0", "bloom-1", "bloom-2", "index"} {
		keys = append(keys, "tempo/tenant/"+blockID.String()+"/"+name)
	}

	bucket, server := newFakeBucket(t, slices.Concat(keys, otherKeys)...)
	cfg := fakeConfig(server)
	cfg.Prefix = "tempo"
	_, _, c, err := NewNoConfirm(cfg)
	require.NoError(t, err)

	require.NoError(t, c.ClearBlock(blockID, "tenant"))
	assert.ElementsMatch(t, otherKeys, bucket.remainingKeys())
	assert.Greater(t, len(bucket.listQueries()), 1, "expected the listing to span several pages")

	require.ErrorIs(t, c.ClearBlock(blockID, ""), backend.ErrEmptyTenantID)
	require.ErrorIs(t, c.ClearBlock(uuid.Nil, "tenant"), backend.ErrEmptyBlockID)
}

func TestMarkBlockCompacted(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	meta := "tenant/" + blockID.String() + "/meta.json"
	compactedMeta := "tenant/" + blockID.String() + "/meta.compacted.json"

	bucket, server := newFakeBucket(t, meta)
	_, _, c, err := NewNoConfirm(fakeConfig(server))
	require.NoError(t, err)

	// the fake bucket only accepts the copy when it's signed with IAM credentials
	require.NoError(t, c.MarkBlockCompacted(blockID, "tenant"))
	assert.ElementsMatch(t, []string{compactedMeta}, bucket.remainingKeys())
	assert.Empty(t, bucket.rejectedRequests())
}

func TestListingErrors(t *testing.T) {
	ctx := context.Background()
	bucket, server := newFakeBucket(t, "tenant/11111111-1111-1111-1111-111111111111/meta.json")
	cfg := fakeConfig(server)
	r, _, c, err := NewNoConfirm(cfg)
	require.NoError(t, err)

	// requests with the failing method fail, other than CreateSession
	var failing string
	bucket.update(func(b *fakeBucket) {
		b.fail = func(r *http.Request) *fakeError {
			if r.Method == failing && !r.URL.Query().Has("session") {
				return &fakeError{status: http.StatusBadRequest, code: "InvalidRequest", message: "the request failed"}
			}
			return nil
		}
	})

	bucket.update(func(*fakeBucket) { failing = http.MethodGet })
	_, err = r.List(ctx, nil)
	require.ErrorContains(t, err, "the request failed")
	_, _, _, err = r.ListBlocks(ctx, "tenant")
	require.ErrorContains(t, err, "the request failed")
	require.ErrorContains(t, c.ClearBlock(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "tenant"), "the request failed")
	_, _, _, err = New(cfg)
	require.ErrorContains(t, err, "unexpected error from ListObjectsV2")

	bucket.update(func(*fakeBucket) { failing = http.MethodDelete })
	require.ErrorContains(t, c.ClearBlock(uuid.MustParse("11111111-1111-1111-1111-111111111111"), "tenant"), "error deleting obj")
}

func TestHedgedRequests(t *testing.T) {
	ctx := context.Background()
	bucket, server := newFakeBucket(t)
	// holds back the first request, which is hedged
	bucket.update(func(b *fakeBucket) { b.getDelay = 10 * time.Second })

	cfg := fakeConfig(server)
	cfg.HedgeRequestsAt = 50 * time.Millisecond
	cfg.HedgeRequestsUpTo = 2
	r, w, _, err := NewNoConfirm(cfg)
	require.NoError(t, err)
	require.NoError(t, w.Write(ctx, "object", backend.KeyPath{"tenant"}, strings.NewReader("data"), 4, nil))
	sessions := bucket.createdSessions()

	buffer := make([]byte, 4)
	require.NoError(t, r.ReadRange(ctx, "object", backend.KeyPath{"tenant"}, 0, buffer, nil))
	assert.Equal(t, "data", string(buffer))

	// the hedged request answered, both were signed with the one session of the hedged client
	assert.Equal(t, 2, bucket.getRequests("tenant/object"))
	assert.Equal(t, sessions+1, bucket.createdSessions())
	assert.Empty(t, bucket.rejectedRequests())
}

func TestRejectedSessionsAreReplaced(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		code          string
		expectedError bool
	}{
		{
			// minio-go retries the request, with a new session
			name:   "expired session",
			status: http.StatusBadRequest,
			code:   "ExpiredToken",
		},
		{
			// the request fails, the next one gets a new session
			name:          "revoked session",
			status:        http.StatusForbidden,
			code:          "AccessDenied",
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			bucket, server := newFakeBucket(t, "tenant-1/index.json.gz")
			r, _, _, err := NewNoConfirm(fakeConfig(server))
			require.NoError(t, err)

			_, err = r.List(ctx, nil)
			require.NoError(t, err)
			require.Equal(t, 1, bucket.createdSessions())

			bucket.rejectSessions(tc.status, tc.code)
			_, err = r.List(ctx, nil)
			if tc.expectedError {
				var errResp minio.ErrorResponse
				require.ErrorAs(t, err, &errResp)
				assert.Equal(t, tc.code, errResp.Code)
			} else {
				require.NoError(t, err)
			}

			tenants, err := r.List(ctx, nil)
			require.NoError(t, err)
			assert.Equal(t, []string{"tenant-1"}, tenants)
			assert.Equal(t, 2, bucket.createdSessions())
		})
	}
}

func TestRequestHosts(t *testing.T) {
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
			bucket, server := newFakeBucket(t)
			cfg := &Config{}
			cfg.Bucket = tc.bucket
			cfg.Region = tc.region
			cfg.ForcePathStyle = tc.forcePathStyle
			cfg.AccessKey = iamAccessKey
			cfg.SecretKey = flagext.SecretWithValue(iamSecretKey)

			_, err := internalNew(cfg, true, redirectTransport{server: server})
			require.NoError(t, err)
			// CreateSession and the listing
			assert.Equal(t, []string{tc.expectedHost, tc.expectedHost}, bucket.requestHosts())
			assert.Empty(t, bucket.rejectedRequests())
		})
	}
}

// redirectTransport sends every request to server, whatever host it's addressed to.
type redirectTransport struct {
	server *httptest.Server
}

func (r redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Host = r.server.Listener.Addr().String()
	return r.server.Client().Transport.RoundTrip(req)
}

func TestBucketEndpoint(t *testing.T) {
	const zonalEndpoint = "s3express-euc1-ist1-az1.eu-central-1.amazonaws.com"

	tests := []struct {
		name             string
		endpoint         string
		lookup           minio.BucketLookupType
		forcePathStyle   bool
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
			name:             "forced path style",
			forcePathStyle:   true,
			expectedEndpoint: zonalEndpoint,
			expectedLookup:   minio.BucketLookupPath,
			expectedURL:      "https://" + zonalEndpoint + "/" + directoryBucketName + "/",
		},
		{
			name:             "other endpoint",
			endpoint:         "localhost:9000",
			expectedEndpoint: "localhost:9000",
			expectedLookup:   minio.BucketLookupPath,
			expectedURL:      "https://localhost:9000/" + directoryBucketName + "/",
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
			cfg := &Config{}
			cfg.Bucket = directoryBucketName
			cfg.Region = "eu-central-1"
			cfg.Endpoint = tc.endpoint
			cfg.BucketLookupType = int(tc.lookup)
			cfg.ForcePathStyle = tc.forcePathStyle

			endpoint, lookup, bucketURL := bucketEndpoint(cfg)
			assert.Equal(t, tc.expectedEndpoint, endpoint)
			assert.Equal(t, tc.expectedLookup, lookup)
			assert.Equal(t, tc.expectedURL, bucketURL.String())
		})
	}
}

func TestZonalEndpoint(t *testing.T) {
	tests := []struct {
		bucket   string
		expected string
	}{
		{bucket: "tempo-traces--euc1-ist1-az1--x-s3", expected: "s3express-euc1-ist1-az1.eu-central-1.amazonaws.com"},
		{bucket: "tempo--euc1-az1--x-s3", expected: "s3express-euc1-az1.eu-central-1.amazonaws.com"},
		{bucket: "tempo--x-s3", expected: ""},
		{bucket: "blerg", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.bucket, func(t *testing.T) {
			assert.Equal(t, tc.expected, zonalEndpoint(tc.bucket, "eu-central-1"))
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
			assert.Equal(t, tc.expected, IsDirectoryBucket(tc.bucket))
		})
	}
}
