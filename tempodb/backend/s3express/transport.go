package s3express

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/signer"
)

const (
	// emptySHA256 is the SHA-256 of an empty payload.
	emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// sessionLeeway is how long before it expires a session is replaced. Sessions last five
	// minutes.
	sessionLeeway = time.Minute

	// maxCreateSessionResponse is how much of a CreateSession response is read.
	maxCreateSessionResponse = 1 << 20
)

// createSessionResult uses the S3 namespace, as AWS does. minio-go only decodes CreateSession
// responses in that namespace.
type createSessionResult struct {
	XMLName     xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CreateSessionResult"`
	Credentials struct {
		AccessKeyID     string    `xml:"AccessKeyId"`
		SecretAccessKey string    `xml:"SecretAccessKey"`
		SessionToken    string    `xml:"SessionToken"`
		Expiration      time.Time `xml:"Expiration"`
	} `xml:"Credentials"`
}

// session holds the credentials of an S3 Express session, and the CreateSession response they came
// in, which minio-go's own CreateSession requests are answered with.
type session struct {
	credentials.Value
	response []byte
}

// transport authenticates requests to an S3 directory bucket the way the AWS SDKs do: it creates a
// session with CreateSession, using the IAM credentials, and signs requests with the session
// credentials for the s3express service, passing the session token in x-amz-s3session-token.
// CopyObject doesn't accept session credentials and is signed with the IAM credentials instead.
//
// minio-go signs requests for the s3 service, and only uses sessions for directory buckets in some
// Availability Zones, so the transport replaces the signature of every request it sends. minio-go
// also sends the requests for those buckets to the S3 Regional endpoint, so the transport sends
// every request to the bucket's endpoint.
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-express-create-session.html
type transport struct {
	next       http.RoundTripper
	creds      *credentials.Credentials
	region     string
	host       string
	sessionURL *url.URL
	now        func() time.Time

	// mtx makes concurrent requests wait for the session one of them creates, instead of each
	// creating its own
	mtx     sync.Mutex
	session session
}

// newTransport returns a transport sending requests through next to the directory bucket at
// bucketURL.
func newTransport(next http.RoundTripper, creds *credentials.Credentials, region string, bucketURL *url.URL) *transport {
	sessionURL := *bucketURL
	sessionURL.RawQuery = "session"

	return &transport{
		next:       next,
		creds:      creds,
		region:     region,
		host:       bucketURL.Host,
		sessionURL: &sessionURL,
		now:        time.Now,
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
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

	signed := req.Clone(req.Context())
	signed.URL.Host = t.host
	signed.Host = t.host
	for _, h := range []string{"Authorization", "X-Amz-Date", "X-Amz-Security-Token", "X-Amz-S3session-Token"} {
		signed.Header.Del(h)
	}

	if signed.Header.Get("X-Amz-Copy-Source") != "" {
		return t.next.RoundTrip(signer.SignV4Express(*signed, iam.AccessKeyID, iam.SecretAccessKey, iam.SessionToken, t.region))
	}

	s, err := t.getSession(req.Context(), iam)
	if err != nil {
		closeBody(req)
		return nil, err
	}
	signed.Header.Set("X-Amz-S3session-Token", s.SessionToken)

	resp, err := t.next.RoundTrip(signer.SignV4Express(*signed, s.AccessKeyID, s.SecretAccessKey, s.SessionToken, t.region))
	if err == nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden) {
		// The session may have expired early, for example if the local clock is behind, or been
		// revoked. The next request creates a new one, which also covers minio-go retrying an
		// ExpiredToken error. Other errors, like a missing object, aren't about the session.
		t.dropSession(s.SessionToken)
	}
	return resp, err
}

// getSession returns the current session, creating a new session when the current one is about to
// expire.
func (t *transport) getSession(ctx context.Context, iam credentials.Value) (session, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.session.Expiration.After(t.now().Add(sessionLeeway)) {
		return t.session, nil
	}

	s, err := t.createSession(ctx, iam)
	if err != nil {
		return session{}, err
	}
	t.session = s
	return s, nil
}

// dropSession forgets the session with token, unless another request replaced it already, so the
// next request creates a new one.
func (t *transport) dropSession(token string) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.session.SessionToken == token {
		t.session = session{}
	}
}

func (t *transport) createSession(ctx context.Context, iam credentials.Value) (session, error) {
	sessionURL := *t.sessionURL
	req := (&http.Request{
		Method: http.MethodGet,
		URL:    &sessionURL,
		Host:   sessionURL.Host,
		Header: http.Header{
			"X-Amz-Content-Sha256":      []string{emptySHA256},
			"X-Amz-Create-Session-Mode": []string{string(minio.SessionReadWrite)},
		},
	}).WithContext(ctx)

	resp, err := t.next.RoundTrip(signer.SignV4Express(*req, iam.AccessKeyID, iam.SecretAccessKey, iam.SessionToken, t.region))
	if err != nil {
		return session{}, fmt.Errorf("error from CreateSession: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCreateSessionResponse))
	if err != nil {
		return session{}, fmt.Errorf("error reading CreateSession response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		errResp := minio.ErrorResponse{StatusCode: resp.StatusCode}
		if err := xml.Unmarshal(body, &errResp); err != nil || errResp.Code == "" {
			errResp.Code = resp.Status
		}
		return session{}, fmt.Errorf("error from CreateSession on %s: %s: %w", t.sessionURL, errResp.Code, errResp)
	}

	var result createSessionResult
	if err := xml.Unmarshal(body, &result); err != nil {
		return session{}, fmt.Errorf("error decoding CreateSession response: %w", err)
	}
	c := result.Credentials
	if c.AccessKeyID == "" || c.SecretAccessKey == "" || c.SessionToken == "" || c.Expiration.IsZero() {
		return session{}, errors.New("error decoding CreateSession response: no session credentials")
	}

	return session{
		Value: credentials.Value{
			AccessKeyID:     c.AccessKeyID,
			SecretAccessKey: c.SecretAccessKey,
			SessionToken:    c.SessionToken,
			Expiration:      c.Expiration,
			SignerType:      credentials.SignatureV4,
		},
		response: body,
	}, nil
}

// sessionResponse answers a CreateSession request of minio-go with the response that created the
// current session.
func (t *transport) sessionResponse(req *http.Request, iam credentials.Value) (*http.Response, error) {
	s, err := t.getSession(req.Context(), iam)
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
		Body:          io.NopCloser(bytes.NewReader(s.response)),
		ContentLength: int64(len(s.response)),
		Request:       req,
	}, nil
}

func closeBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}
