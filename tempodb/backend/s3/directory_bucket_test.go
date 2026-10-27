package s3

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/flagext"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// directoryBucketName follows the naming scheme of S3 directory buckets, here one in a Local Zone.
const directoryBucketName = "tempo--euc1-ist1-az1--x-s3"

// TestDirectoryBucketReadWrite writes, reads, lists, compacts and clears a block in a directory
// bucket. It runs against a fake directory bucket, and against a real one when
// TEMPO_TEST_S3_DIRECTORY_BUCKET and TEMPO_TEST_S3_DIRECTORY_BUCKET_REGION are set, with the
// credentials of the environment, for example AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.
func TestDirectoryBucketReadWrite(t *testing.T) {
	t.Run("fake", func(t *testing.T) {
		_, endpoint := newFakeTLSBucket(t)
		cfg := fakeConfig(directoryBucketName, endpoint)
		cfg.Prefix = "tempo"
		cfg.Insecure = false
		cfg.InsecureSkipVerify = true
		testDirectoryBucketReadWrite(t, cfg)
	})

	t.Run("aws", func(t *testing.T) {
		bucket, region := os.Getenv("TEMPO_TEST_S3_DIRECTORY_BUCKET"), os.Getenv("TEMPO_TEST_S3_DIRECTORY_BUCKET_REGION")
		if bucket == "" || region == "" {
			t.Skip("set TEMPO_TEST_S3_DIRECTORY_BUCKET and TEMPO_TEST_S3_DIRECTORY_BUCKET_REGION to run against an S3 directory bucket")
		}
		testDirectoryBucketReadWrite(t, &Config{
			Bucket: bucket,
			Region: region,
			// keeps the test away from any other data in the bucket
			Prefix: "tempo-test-" + uuid.NewString(),
		})
	})
}

func testDirectoryBucketReadWrite(t *testing.T, cfg *Config) {
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
		bucket         string
		prefix         string
		expectedV2     bool
		expectedPrefix string
	}{
		{name: "general purpose bucket", bucket: "blerg", prefix: "tempo", expectedPrefix: "tempo"},
		{name: "directory bucket without prefix", bucket: directoryBucketName, expectedV2: true, expectedPrefix: ""},
		{name: "directory bucket with prefix", bucket: directoryBucketName, prefix: "tempo", expectedV2: true, expectedPrefix: "tempo/"},
		{name: "directory bucket with trailing slash prefix", bucket: directoryBucketName, prefix: "a/b/c/", expectedV2: true, expectedPrefix: "a/b/c/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, isDirectoryBucket(tc.bucket))
			cfg := fakeConfig(tc.bucket, endpoint)
			cfg.Prefix = tc.prefix
			_, _, _, err := New(cfg)
			require.NoError(t, err)

			queries := bucket.listQueries()
			require.Len(t, queries, 1)
			assert.Equal(t, tc.expectedV2, queries[0].Get("list-type") == "2")
			assert.Equal(t, tc.expectedPrefix, queries[0].Get("prefix"))
		})
	}
}

func TestList(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111").String()
	keys := []string{
		"tempo/tempo_cluster_seed.json",
		"tempo/tenant-1/index.json.gz",
		"tempo/tenant-2/" + blockID + "/meta.json",
		"tempo/tenant-3/" + blockID + "/data.parquet",
	}

	tests := []struct {
		name       string
		bucket     string
		expectedV2 bool
	}{
		{name: "general purpose bucket", bucket: "blerg"},
		{name: "directory bucket", bucket: directoryBucketName, expectedV2: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, isDirectoryBucket(tc.bucket), keys...)
			cfg := fakeConfig(tc.bucket, endpoint)
			cfg.Prefix = "tempo"
			r, _, _, err := NewNoConfirm(cfg)
			require.NoError(t, err)

			tenants, err := r.List(context.Background(), nil)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"tenant-1", "tenant-2", "tenant-3"}, tenants)

			queries := bucket.listQueries()
			assert.Greater(t, len(queries), 1, "expected the listing to span several pages")
			for _, q := range queries {
				assert.Equal(t, tc.expectedV2, q.Get("list-type") == "2")
			}
		})
	}
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
			expectedStartAfter: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket, endpoint := newFakeBucket(t, isDirectoryBucket(tc.bucket), keys...)
			cfg := fakeConfig(tc.bucket, endpoint)
			cfg.Prefix = "tempo"
			cfg.ListBlocksConcurrency = 3
			r, _, _, err := NewNoConfirm(cfg)
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

// fakeConfig returns the config of a backend using the fake bucket at endpoint over http.
func fakeConfig(bucket, endpoint string) *Config {
	return &Config{
		Region:    "blerg",
		AccessKey: "test",
		SecretKey: flagext.SecretWithValue("test"),
		Bucket:    bucket,
		Insecure:  true,
		Endpoint:  endpoint,
	}
}

// fakeBucket is an in-memory S3 bucket serving the requests of the backend. It returns at most two
// entries per list page. With directory set it follows the rules of S3 directory buckets: only
// ListObjectsV2 is supported, without start-after, prefixes must end in "/", keys aren't listed in
// lexicographical order, multipart uploads need consecutive part numbers, MD5 checksums aren't
// supported, and requests must be signed with an S3 Express session, except for CreateSession and
// CopyObject which take IAM credentials. Requests can be path or virtual-hosted style.
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-differences.html
type fakeBucket struct {
	directory       bool
	sessionLifetime time.Duration
	sessionDelay    time.Duration
	now             func() time.Time
	// omitNextMarker leaves NextMarker out of V1 listings, like some S3-compatible stores do
	omitNextMarker bool

	mtx          sync.Mutex
	objects      map[string][]byte
	uploads      map[string]map[int][]byte
	lastUploadID int
	queries      []url.Values
	hosts        []string
	sessions     int
}

const (
	fakeSessionAccessKey = "session-access-key"
	fakeSessionToken     = "session-token"
)

// fakeModTime is when every object of a fakeBucket was last modified.
var fakeModTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

type fakeListResult struct {
	XMLName               xml.Name           `xml:"ListBucketResult"`
	IsTruncated           bool               `xml:"IsTruncated"`
	NextMarker            string             `xml:"NextMarker,omitempty"`
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

// newFakeBucket returns a fake bucket served over http, holding empty objects at keys, and its
// endpoint.
func newFakeBucket(t *testing.T, directory bool, keys ...string) (*fakeBucket, string) {
	t.Helper()

	b := newFake(directory, keys...)
	server := testServer(t, b.serveHTTP)
	return b, server.URL[7:] // [7:] -> strip http://
}

// newFakeTLSBucket returns an empty fake directory bucket served over https, and its endpoint.
// Writes need https: over http, minio-go signs uploads with streaming signatures, which
// s3ExpressTransport can't sign for S3 Express.
func newFakeTLSBucket(t *testing.T) (*fakeBucket, string) {
	t.Helper()

	b := newFake(true)
	server := httptest.NewTLSServer(http.HandlerFunc(b.serveHTTP))
	t.Cleanup(server.Close)
	return b, server.URL[8:] // [8:] -> strip https://
}

func newFake(directory bool, keys ...string) *fakeBucket {
	b := &fakeBucket{
		directory:       directory,
		sessionLifetime: 5 * time.Minute,
		now:             time.Now,
		objects:         map[string][]byte{},
		uploads:         map[string]map[int][]byte{},
	}
	for _, key := range keys {
		b.objects[key] = nil
	}
	return b
}

func (b *fakeBucket) listQueries() []url.Values {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return slices.Clone(b.queries)
}

func (b *fakeBucket) requestHosts() []string {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return slices.Clone(b.hosts)
}

func (b *fakeBucket) remainingKeys() []string {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return slices.Collect(maps.Keys(b.objects))
}

func (b *fakeBucket) createdSessions() int {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.sessions
}

func (b *fakeBucket) serveHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	bucket, key := requestPath(r)
	isCreateSession := r.Method == http.MethodGet && key == "" && query.Has("session")
	if isCreateSession {
		// outside of the lock, so concurrent CreateSession requests overlap
		time.Sleep(b.sessionDelay)
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.hosts = append(b.hosts, r.Host)

	if isCreateSession {
		b.createSession(w, r)
		return
	}
	if err := b.authorize(r); err != nil {
		writeS3Error(w, http.StatusForbidden, "AccessDenied", err.Error())
		return
	}
	if b.directory && r.Header.Get("Content-Md5") != "" {
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "MD5 checksums aren't supported by directory buckets")
		return
	}

	switch {
	case r.Method == http.MethodGet && key == "":
		b.list(w, query)
	case r.Method == http.MethodPost && query.Has("uploads"):
		b.createMultipartUpload(w, key)
	case r.Method == http.MethodPut && query.Has("uploadId"):
		b.uploadPart(w, r, query)
	case r.Method == http.MethodPost && query.Has("uploadId"):
		b.completeMultipartUpload(w, r, bucket, key, query.Get("uploadId"))
	case r.Method == http.MethodPut && r.Header.Get("X-Amz-Copy-Source") != "":
		b.copyObject(w, r, key)
	case r.Method == http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeS3Error(w, http.StatusBadRequest, "IncompleteBody", err.Error())
			return
		}
		b.objects[key] = body
		w.Header().Set("ETag", `"etag"`)
	case r.Method == http.MethodGet || r.Method == http.MethodHead:
		b.getObject(w, r, key)
	case r.Method == http.MethodDelete:
		delete(b.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// requestPath returns the bucket and the key of the object r is for, the key is empty for requests
// to the bucket. Requests whose host starts with the name of a directory bucket are virtual-hosted
// style.
func requestPath(r *http.Request) (bucket, key string) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if bucket, _, _ := strings.Cut(r.Host, "."); isDirectoryBucket(bucket) {
		return bucket, path
	}
	// path-style requests: /<bucket>/<key>
	bucket, key, _ = strings.Cut(path, "/")
	return bucket, key
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
		writeS3Error(w, http.StatusForbidden, "AccessDenied", "CreateSession must be signed with IAM credentials for s3express")
		return
	}
	b.sessions++

	var res createSessionResult
	res.Credentials.AccessKeyID = fakeSessionAccessKey
	res.Credentials.SecretAccessKey = "session-secret-key"
	res.Credentials.SessionToken = fakeSessionToken
	res.Credentials.Expiration = b.now().Add(b.sessionLifetime)
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

	v2 := query.Get("list-type") == "2"
	prefix, delimiter := query.Get("prefix"), query.Get("delimiter")
	switch {
	case b.directory && !v2:
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "This bucket does not support ListObjects API. Consider using ListObjectsV2 API.")
		return
	case b.directory && query.Has("start-after"):
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "start-after is not supported by directory buckets")
		return
	case b.directory && prefix != "" && !strings.HasSuffix(prefix, "/"):
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "directory buckets only support prefixes that end in a delimiter")
		return
	}

	// list the entries after start-after (V2) or marker (V1)
	after := query.Get("start-after")
	if !v2 {
		after = query.Get("marker")
	}

	type entry struct {
		name     string
		isPrefix bool
	}
	var (
		entries []entry
		seen    = map[string]struct{}{}
	)
	for key := range b.objects {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		e := entry{name: key}
		if i := strings.Index(key[len(prefix):], delimiter); delimiter != "" && i >= 0 {
			e = entry{name: key[:len(prefix)+i+len(delimiter)], isPrefix: true}
		}
		if _, ok := seen[e.name]; ok || (after != "" && e.name <= after) {
			continue
		}
		seen[e.name] = struct{}{}
		entries = append(entries, e)
	}
	slices.SortFunc(entries, func(a, c entry) int {
		if b.directory {
			return strings.Compare(c.name, a.name)
		}
		return strings.Compare(a.name, c.name)
	})

	// V2 continuation tokens are the offset of the next page, V1 pages continue after their last entry
	pageSize := 2
	if maxKeys, _ := strconv.Atoi(query.Get("max-keys")); maxKeys > 0 && maxKeys < pageSize {
		pageSize = maxKeys
	}
	start, _ := strconv.Atoi(query.Get("continuation-token"))
	start = min(start, len(entries))
	end := min(start+pageSize, len(entries))

	res := fakeListResult{IsTruncated: end < len(entries)}
	switch {
	case res.IsTruncated && v2:
		res.NextContinuationToken = strconv.Itoa(end)
	case res.IsTruncated && !b.omitNextMarker:
		res.NextMarker = entries[end-1].name
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

func (b *fakeBucket) getObject(w http.ResponseWriter, r *http.Request, key string) {
	data, ok := b.objects[key]
	if !ok {
		writeS3Error(w, http.StatusNotFound, minio.NoSuchKey, "The specified key does not exist.")
		return
	}
	w.Header().Set("ETag", `"etag"`)
	http.ServeContent(w, r, key, fakeModTime, bytes.NewReader(data))
}

func (b *fakeBucket) copyObject(w http.ResponseWriter, r *http.Request, key string) {
	source, _ := url.PathUnescape(r.Header.Get("X-Amz-Copy-Source"))
	_, sourceKey, _ := strings.Cut(strings.TrimPrefix(source, "/"), "/")
	data, ok := b.objects[sourceKey]
	if !ok {
		writeS3Error(w, http.StatusNotFound, minio.NoSuchKey, "The specified key does not exist.")
		return
	}
	b.objects[key] = data
	_, _ = w.Write([]byte(`<CopyObjectResult><ETag>"etag"</ETag></CopyObjectResult>`))
}

func (b *fakeBucket) createMultipartUpload(w http.ResponseWriter, key string) {
	b.lastUploadID++
	uploadID := strconv.Itoa(b.lastUploadID)
	b.uploads[uploadID] = map[int][]byte{}
	_ = xml.NewEncoder(w).Encode(struct {
		XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
		Key      string   `xml:"Key"`
		UploadID string   `xml:"UploadId"`
	}{Key: key, UploadID: uploadID})
}

func (b *fakeBucket) uploadPart(w http.ResponseWriter, r *http.Request, query url.Values) {
	parts, ok := b.uploads[query.Get("uploadId")]
	if !ok {
		writeS3Error(w, http.StatusNotFound, "NoSuchUpload", "The specified upload does not exist.")
		return
	}
	partNumber, err := strconv.Atoi(query.Get("partNumber"))
	if err != nil {
		writeS3Error(w, http.StatusBadRequest, "InvalidArgument", err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeS3Error(w, http.StatusBadRequest, "IncompleteBody", err.Error())
		return
	}
	parts[partNumber] = body
	w.Header().Set("ETag", fmt.Sprintf(`"part-%d"`, partNumber))
}

func (b *fakeBucket) completeMultipartUpload(w http.ResponseWriter, r *http.Request, bucket, key, uploadID string) {
	parts, ok := b.uploads[uploadID]
	if !ok {
		writeS3Error(w, http.StatusNotFound, "NoSuchUpload", "The specified upload does not exist.")
		return
	}
	var req struct {
		Parts []struct {
			PartNumber int `xml:"PartNumber"`
		} `xml:"Part"`
	}
	if err := xml.NewDecoder(r.Body).Decode(&req); err != nil {
		writeS3Error(w, http.StatusBadRequest, "MalformedXML", err.Error())
		return
	}

	var data []byte
	for i, part := range req.Parts {
		if b.directory && part.PartNumber != i+1 {
			writeS3Error(w, http.StatusBadRequest, "InvalidPartOrder", "directory buckets require consecutive part numbers")
			return
		}
		p, ok := parts[part.PartNumber]
		if !ok {
			writeS3Error(w, http.StatusBadRequest, "InvalidPart", "part "+strconv.Itoa(part.PartNumber)+" wasn't uploaded")
			return
		}
		data = append(data, p...)
	}
	b.objects[key] = data
	delete(b.uploads, uploadID)
	_ = xml.NewEncoder(w).Encode(struct {
		XMLName xml.Name `xml:"CompleteMultipartUploadResult"`
		Bucket  string   `xml:"Bucket"`
		Key     string   `xml:"Key"`
		ETag    string   `xml:"ETag"`
	}{Bucket: bucket, Key: key, ETag: `"etag"`})
}

func writeS3Error(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = xml.NewEncoder(w).Encode(minio.ErrorResponse{Code: code, Message: message})
}
