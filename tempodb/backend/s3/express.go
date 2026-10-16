package s3

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/signer"
)

const (
	// emptySHA256 is the SHA-256 of an empty payload.
	emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// s3ExpressSessionLeeway is how long before it expires a session is replaced. Sessions last
	// five minutes.
	s3ExpressSessionLeeway = time.Minute
)

// isDirectoryBucket reports whether bucket names an S3 directory bucket, in an Availability Zone
// (S3 Express One Zone) or a Local Zone, or an access point for one. These are identified by the
// same name suffixes the AWS SDKs use.
func isDirectoryBucket(bucket string) bool {
	return strings.HasSuffix(bucket, "--x-s3") || strings.HasSuffix(bucket, "--xa-s3")
}

// s3ExpressZonalEndpoint returns the Zonal endpoint of a directory bucket, built from the zone ID
// in its name (<base>--<zone-id>--x-s3) like the AWS SDKs do.
func s3ExpressZonalEndpoint(bucket, region string) string {
	parts := strings.Split(bucket, "--")
	if len(parts) < 3 {
		return ""
	}
	return "s3express-" + parts[len(parts)-2] + "." + region + ".amazonaws.com"
}

// createSessionResult uses the S3 namespace, as AWS does. minio-go only decodes CreateSession
// responses in that namespace, including those sessionResponse answers its requests with.
type createSessionResult struct {
	XMLName     xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CreateSessionResult"`
	Credentials struct {
		AccessKeyID     string    `xml:"AccessKeyId"`
		SecretAccessKey string    `xml:"SecretAccessKey"`
		SessionToken    string    `xml:"SessionToken"`
		Expiration      time.Time `xml:"Expiration"`
	} `xml:"Credentials"`
}

// s3ExpressTransport authenticates requests to an S3 directory bucket the way the AWS SDKs do:
// it creates a session with CreateSession, using the IAM credentials, and signs requests with the
// session credentials for the s3express service, passing the session token in
// x-amz-s3session-token. CopyObject doesn't accept session credentials and is signed with the IAM
// credentials instead.
//
// minio-go signs requests for the s3 service, and only uses sessions for directory buckets in some
// Availability Zones, so the transport replaces the signature of every request it sends.
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-create-session.html
type s3ExpressTransport struct {
	next       http.RoundTripper
	creds      *credentials.Credentials
	region     string
	sessionURL string
	now        func() time.Time

	mtx     sync.Mutex
	session credentials.Value
}

func newS3ExpressTransport(next http.RoundTripper, creds *credentials.Credentials, region, sessionURL string) *s3ExpressTransport {
	return &s3ExpressTransport{
		next:       next,
		creds:      creds,
		region:     region,
		sessionURL: sessionURL,
		now:        time.Now,
	}
}

func (t *s3ExpressTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	iam, err := t.creds.GetWithContext(&credentials.CredContext{Client: http.DefaultClient})
	if err != nil {
		closeBody(req)
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}
	if iam.AccessKeyID == "" || iam.SignerType.IsAnonymous() {
		return t.next.RoundTrip(req)
	}

	// minio-go creates its own sessions for some directory buckets, answer those from ours
	if req.URL.Query().Has("session") {
		closeBody(req)
		return t.sessionResponse(req, iam)
	}

	if strings.HasPrefix(req.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
		closeBody(req)
		return nil, errors.New("S3 directory buckets require https, streaming signatures used over http can't be signed for S3 Express")
	}

	signed := req.Clone(req.Context())
	for _, h := range []string{"Authorization", "X-Amz-Date", "X-Amz-Security-Token", "X-Amz-S3session-Token"} {
		signed.Header.Del(h)
	}

	if signed.Header.Get("X-Amz-Copy-Source") != "" {
		return t.next.RoundTrip(signer.SignV4Express(*signed, iam.AccessKeyID, iam.SecretAccessKey, iam.SessionToken, t.region))
	}

	session, err := t.getSession(req.Context(), iam)
	if err != nil {
		closeBody(req)
		return nil, err
	}
	signed.Header.Set("X-Amz-S3session-Token", session.SessionToken)
	return t.next.RoundTrip(signer.SignV4Express(*signed, session.AccessKeyID, session.SecretAccessKey, session.SessionToken, t.region))
}

// getSession returns the current session credentials, creating a new session when the current
// one is about to expire.
func (t *s3ExpressTransport) getSession(ctx context.Context, iam credentials.Value) (credentials.Value, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.session.Expiration.After(t.now().Add(s3ExpressSessionLeeway)) {
		return t.session, nil
	}

	session, err := t.createSession(ctx, iam)
	if err != nil {
		return credentials.Value{}, err
	}
	t.session = session
	return session, nil
}

func (t *s3ExpressTransport) createSession(ctx context.Context, iam credentials.Value) (credentials.Value, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.sessionURL, nil)
	if err != nil {
		return credentials.Value{}, fmt.Errorf("failed to create CreateSession request: %w", err)
	}
	req.Header.Set("X-Amz-Content-Sha256", emptySHA256)
	req.Header.Set("X-Amz-Create-Session-Mode", string(minio.SessionReadWrite))

	resp, err := t.next.RoundTrip(signer.SignV4Express(*req, iam.AccessKeyID, iam.SecretAccessKey, iam.SessionToken, t.region))
	if err != nil {
		return credentials.Value{}, fmt.Errorf("error from CreateSession: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return credentials.Value{}, fmt.Errorf("error reading CreateSession response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		errResp := minio.ErrorResponse{StatusCode: resp.StatusCode}
		if err := xml.Unmarshal(body, &errResp); err != nil || errResp.Code == "" {
			errResp.Code = resp.Status
		}
		return credentials.Value{}, fmt.Errorf("error from CreateSession on %s: %s: %w", t.sessionURL, errResp.Code, errResp)
	}

	var result createSessionResult
	if err := xml.Unmarshal(body, &result); err != nil {
		return credentials.Value{}, fmt.Errorf("error decoding CreateSession response: %w", err)
	}
	return credentials.Value{
		AccessKeyID:     result.Credentials.AccessKeyID,
		SecretAccessKey: result.Credentials.SecretAccessKey,
		SessionToken:    result.Credentials.SessionToken,
		Expiration:      result.Credentials.Expiration,
		SignerType:      credentials.SignatureV4,
	}, nil
}

func (t *s3ExpressTransport) sessionResponse(req *http.Request, iam credentials.Value) (*http.Response, error) {
	session, err := t.getSession(req.Context(), iam)
	if err != nil {
		return nil, err
	}

	var result createSessionResult
	result.Credentials.AccessKeyID = session.AccessKeyID
	result.Credentials.SecretAccessKey = session.SecretAccessKey
	result.Credentials.SessionToken = session.SessionToken
	result.Credentials.Expiration = session.Expiration
	body, err := xml.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{"application/xml"}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}, nil
}

func closeBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}
