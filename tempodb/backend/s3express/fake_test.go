package s3express

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/dskit/flagext"
	minio "github.com/minio/minio-go/v7"
)

const (
	// directoryBucketName follows the naming scheme of S3 directory buckets, here one in a Local
	// Zone.
	directoryBucketName = "tempo--euc1-ist1-az1--x-s3"

	// the IAM credentials the fake bucket accepts
	iamAccessKey = "test"
	iamSecretKey = "test-secret"
)

// fakeModTime is when every object of a fakeBucket was last modified.
var fakeModTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// fakeBucket is an in-memory S3 directory bucket serving the requests of the backend. It returns at
// most two entries per list page, and follows the rules of directory buckets: only ListObjectsV2 is
// supported, without start-after, prefixes must end in "/", keys aren't listed in lexicographical
// order, multipart uploads need consecutive part numbers, and MD5 checksums aren't supported.
// Requests must be signed for the s3express service with the credentials of a session it created,
// except for CreateSession and CopyObject, which take IAM credentials. It recomputes the signature of
// every request. Requests can be path or virtual-hosted style.
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-differences.html
type fakeBucket struct {
	// mtx guards the settings, which update changes, as well as the state
	mtx sync.Mutex

	sessionLifetime time.Duration
	// sessionDelay delays CreateSession, so concurrent requests overlap with it
	sessionDelay time.Duration
	// getDelay delays the first GetObject of each object, so the request is hedged
	getDelay time.Duration
	now      func() time.Time
	// fail returns the error to answer an authorized request with, if any
	fail func(r *http.Request) *fakeError

	objects      map[string][]byte
	uploads      map[string]map[int][]byte
	lastUploadID int
	queries      []url.Values
	hosts        []string
	gets         map[string]int
	sessions     map[string]*fakeSession
	authFailures []string
	createdCount int
}

// fakeSession is a session of a fakeBucket, by its access key.
type fakeSession struct {
	secretKey  string
	token      string
	expiration time.Time
	// rejection is returned for requests signed with the session, as if it had been revoked
	rejection *fakeError
}

type fakeError struct {
	status  int
	code    string
	message string
}

// newFakeBucket returns a fake directory bucket holding empty objects at keys, and the https
// server serving it.
func newFakeBucket(t *testing.T, keys ...string) (*fakeBucket, *httptest.Server) {
	t.Helper()

	b := &fakeBucket{
		sessionLifetime: 5 * time.Minute,
		now:             time.Now,
		objects:         map[string][]byte{},
		uploads:         map[string]map[int][]byte{},
		gets:            map[string]int{},
		sessions:        map[string]*fakeSession{},
	}
	for _, key := range keys {
		b.objects[key] = nil
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(b.serveHTTP))
	// connections the clients abandon, like those of hedged requests, aren't errors here
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	t.Cleanup(server.Close)
	return b, server
}

// fakeConfig returns the config of a backend using the fake bucket served by server.
func fakeConfig(server *httptest.Server) *Config {
	cfg := &Config{}
	cfg.Bucket = directoryBucketName
	cfg.Region = "eu-central-1"
	cfg.Endpoint = server.Listener.Addr().String()
	cfg.AccessKey = iamAccessKey
	cfg.SecretKey = flagext.SecretWithValue(iamSecretKey)
	// trust the certificate of the test server
	cfg.InsecureSkipVerify = true
	return cfg
}

// fakeBucketURL returns the URL of the fake bucket served by server, for path-style requests.
func fakeBucketURL(server *httptest.Server) *url.URL {
	return &url.URL{Scheme: "https", Host: server.Listener.Addr().String(), Path: "/" + directoryBucketName + "/"}
}

// update changes the settings of the bucket with f, while it serves requests.
func (b *fakeBucket) update(f func(b *fakeBucket)) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	f(b)
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

func (b *fakeBucket) getRequests(key string) int {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.gets[key]
}

func (b *fakeBucket) createdSessions() int {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.createdCount
}

// rejectedRequests returns the requests the bucket rejected because of their credentials, other
// than those signed with a session rejectSessions rejects.
func (b *fakeBucket) rejectedRequests() []string {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return slices.Clone(b.authFailures)
}

// rejectSessions makes the bucket reject the requests signed with the sessions it created so far,
// with status and code.
func (b *fakeBucket) rejectSessions(status int, code string) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for _, s := range b.sessions {
		s.rejection = &fakeError{status: status, code: code, message: "the session was rejected"}
	}
}

func (b *fakeBucket) serveHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	bucket, key := requestPath(r)
	isCreateSession := r.Method == http.MethodGet && key == "" && query.Has("session")
	// outside of the lock, so concurrent requests overlap
	if !b.wait(r, key, isCreateSession) {
		return
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.hosts = append(b.hosts, r.Host)

	if err := b.authorize(r, isCreateSession); err != nil {
		writeS3Error(w, err.status, err.code, err.message)
		return
	}
	if b.fail != nil {
		if err := b.fail(r); err != nil {
			writeS3Error(w, err.status, err.code, err.message)
			return
		}
	}
	if r.Header.Get("Content-Md5") != "" {
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "MD5 checksums aren't supported by directory buckets")
		return
	}

	switch {
	case isCreateSession:
		b.createSession(w)
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

// wait delays CreateSession by sessionDelay, and the first GetObject of each key by getDelay. It
// returns false when the client gave up on the request meanwhile.
func (b *fakeBucket) wait(r *http.Request, key string, isCreateSession bool) bool {
	var delay time.Duration
	b.mtx.Lock()
	switch {
	case isCreateSession:
		delay = b.sessionDelay
	case r.Method == http.MethodGet && key != "":
		b.gets[key]++
		if b.gets[key] == 1 {
			delay = b.getDelay
		}
	}
	b.mtx.Unlock()

	select {
	case <-time.After(delay):
		return true
	case <-r.Context().Done():
		return false
	}
}

// requestPath returns the bucket and the key of the object r is for, the key is empty for requests
// to the bucket. Requests whose host starts with the name of a directory bucket are virtual-hosted
// style.
func requestPath(r *http.Request) (bucket, key string) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if bucket, _, _ := strings.Cut(r.Host, "."); IsDirectoryBucket(bucket) {
		return bucket, path
	}
	// path-style requests: /<bucket>/<key>
	bucket, key, _ = strings.Cut(path, "/")
	return bucket, key
}

// authorize checks requests are signed like directory buckets expect: CreateSession and CopyObject
// with IAM credentials, other requests with the credentials of a session, which hasn't expired,
// and its token. All of them are signed for the s3express service.
func (b *fakeBucket) authorize(r *http.Request, isCreateSession bool) *fakeError {
	reject := func(code, message string) *fakeError {
		b.authFailures = append(b.authFailures, r.Method+" "+r.URL.String()+": "+message)
		return &fakeError{status: http.StatusForbidden, code: code, message: message}
	}

	accessKey, service := signingScope(r)
	if service != "s3express" {
		return reject("AccessDenied", fmt.Sprintf("expected a signature for s3express, got %q", service))
	}

	secretKey := iamSecretKey
	if isCreateSession || r.Header.Get("X-Amz-Copy-Source") != "" {
		if accessKey != iamAccessKey || r.Header.Get("X-Amz-S3session-Token") != "" {
			return reject("AccessDenied", "CreateSession and CopyObject must be signed with IAM credentials")
		}
	} else {
		s, ok := b.sessions[accessKey]
		switch {
		case !ok || r.Header.Get("X-Amz-S3session-Token") != s.token:
			return reject("AccessDenied", "expected the credentials and token of a session")
		case s.rejection != nil:
			return s.rejection
		case !b.now().Before(s.expiration):
			return reject("ExpiredToken", "the session expired")
		}
		secretKey = s.secretKey
	}

	if err := verifySignature(r, secretKey); err != nil {
		return reject("SignatureDoesNotMatch", err.Error())
	}
	return nil
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

// verifySignature checks r was signed with secretKey, by recomputing its SigV4 signature from the
// request as it was received.
// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
func verifySignature(r *http.Request, secretKey string) error {
	auth, ok := strings.CutPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ")
	if !ok {
		return errors.New("expected an AWS4-HMAC-SHA256 signature")
	}
	fields := map[string]string{}
	for _, field := range strings.Split(auth, ",") {
		k, v, _ := strings.Cut(strings.TrimSpace(field), "=")
		fields[k] = v
	}

	// Credential=<access key>/<date>/<region>/<service>/aws4_request
	_, scope, _ := strings.Cut(fields["Credential"], "/")
	scopeParts := strings.Split(scope, "/")
	if len(scopeParts) != 4 {
		return fmt.Errorf("unexpected credential scope %q", scope)
	}

	var canonicalHeaders strings.Builder
	for _, name := range strings.Split(fields["SignedHeaders"], ";") {
		value := strings.Join(r.Header.Values(name), ",")
		if name == "host" {
			value = r.Host
		}
		canonicalHeaders.WriteString(name + ":" + strings.Join(strings.Fields(value), " ") + "\n")
	}
	payloadHash := r.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = "UNSIGNED-PAYLOAD"
	}
	if payloadHash != "UNSIGNED-PAYLOAD" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("failed to read the body: %w", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		if sum := sha256.Sum256(body); hex.EncodeToString(sum[:]) != payloadHash {
			return errors.New("the body doesn't match X-Amz-Content-Sha256")
		}
	}
	canonicalRequest := strings.Join([]string{
		r.Method,
		r.URL.EscapedPath(),
		strings.ReplaceAll(r.URL.Query().Encode(), "+", "%20"),
		canonicalHeaders.String(),
		fields["SignedHeaders"],
		payloadHash,
	}, "\n")
	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", r.Header.Get("X-Amz-Date"), scope, hex.EncodeToString(canonicalRequestHash[:])}, "\n")

	// the signing key is derived from the secret key and each part of the scope in turn
	key := []byte("AWS4" + secretKey)
	for _, part := range scopeParts {
		key = hmacSHA256(key, part)
	}
	expected := hex.EncodeToString(hmacSHA256(key, stringToSign))
	if !hmac.Equal([]byte(expected), []byte(fields["Signature"])) {
		return errors.New("the signature doesn't match the request")
	}
	return nil
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// createSession serves CreateSession, creating a session with new credentials.
func (b *fakeBucket) createSession(w http.ResponseWriter) {
	b.createdCount++
	n := strconv.Itoa(b.createdCount)
	accessKey := "session-access-key-" + n
	s := &fakeSession{
		secretKey:  "session-secret-key-" + n,
		token:      "session-token-" + n,
		expiration: b.now().Add(b.sessionLifetime).UTC().Truncate(time.Second),
	}
	b.sessions[accessKey] = s

	var res createSessionResult
	res.Credentials.AccessKeyID = accessKey
	res.Credentials.SecretAccessKey = s.secretKey
	res.Credentials.SessionToken = s.token
	res.Credentials.Expiration = s.expiration
	_ = xml.NewEncoder(w).Encode(res)
}

type fakeListResult struct {
	XMLName               xml.Name           `xml:"ListBucketResult"`
	IsTruncated           bool               `xml:"IsTruncated"`
	NextContinuationToken string             `xml:"NextContinuationToken,omitempty"`
	Contents              []fakeListContents `xml:"Contents"`
	CommonPrefixes        []fakeListPrefix   `xml:"CommonPrefixes"`
}

type fakeListContents struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
}

type fakeListPrefix struct {
	Prefix string `xml:"Prefix"`
}

func (b *fakeBucket) list(w http.ResponseWriter, query url.Values) {
	b.queries = append(b.queries, query)

	prefix, delimiter := query.Get("prefix"), query.Get("delimiter")
	switch {
	case query.Get("list-type") != "2":
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "This bucket does not support ListObjects API. Consider using ListObjectsV2 API.")
		return
	case query.Has("start-after"):
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "start-after is not supported by directory buckets")
		return
	case prefix != "" && !strings.HasSuffix(prefix, "/"):
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "directory buckets only support prefixes that end in a delimiter")
		return
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
		if _, ok := seen[e.name]; ok {
			continue
		}
		seen[e.name] = struct{}{}
		entries = append(entries, e)
	}
	// directory buckets don't list keys in lexicographical order, reverse it
	slices.SortFunc(entries, func(a, c entry) int {
		return strings.Compare(c.name, a.name)
	})

	// continuation tokens are the offset of the next page
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
			res.Contents = append(res.Contents, fakeListContents{Key: e.name, LastModified: fakeModTime})
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
		if part.PartNumber != i+1 {
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
