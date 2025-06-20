// Package aws provides AWS_MSK_IAM sasl authentication as specified in the
// Java source.
//
// The Java source can be found at https://github.com/aws/aws-msk-iam-auth.
package aws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/sasl"
)

// Auth contains an AWS AccessKey and SecretKey for authentication.
//
// This client may add fields to this struct in the future if Kafka adds more
// capabilities to MSK IAM.
type Auth struct {
	// AccessKey is an AWS AccessKey.
	AccessKey string

	// AccessKey is an AWS SecretKey.
	SecretKey string

	// SessionToken, if non-empty, is a session / security token to use for
	// authentication.
	//
	// See the following link for more details:
	//
	//     https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html
	//
	SessionToken string

	// UserAgent is the user agent to for the client to use when connecting
	// to Kafka, overriding the default "franz-go/<runtime.Version()>/<hostname>".
	//
	// Setting a UserAgent allows authorizing based on the aws:UserAgent
	// condition key; see the following link for more details:
	//
	//     https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_condition-keys.html#condition-keys-useragent
	//
	UserAgent string

	_ struct{} // require explicit field initialization
}

var hostname, _ = os.Hostname()

func init() {
	if hostname == "" {
		hostname = "unknown"
	}
}

// AsManagedStreamingIAMMechanism returns a sasl mechanism that will use 'a' as
// credentials for all sasl sessions.
//
// This is a shortcut for using the ManagedStreamingIAM function and is useful
// when you do not need to live-rotate credentials.
func (a Auth) AsManagedStreamingIAMMechanism() sasl.Mechanism {
	return ManagedStreamingIAM(func(context.Context) (Auth, error) {
		return a, nil
	})
}

type mskiam func(context.Context) (Auth, error)

// ManagedStreamingIAM returns a sasl mechanism that will call authFn whenever
// sasl authentication is needed. The returned Auth is used for a single
// session.
func ManagedStreamingIAM(authFn func(context.Context) (Auth, error)) sasl.Mechanism {
	return mskiam(authFn)
}

func (mskiam) Name() string { return "AWS_MSK_IAM" }

func (fn mskiam) Authenticate(ctx context.Context, host string) (sasl.Session, []byte, error) {
	auth, err := fn(ctx)
	if err != nil {
		return nil, nil, err
	}

	challenge, err := challenge(auth, host)
	if err != nil {
		return nil, nil, err
	}

	return new(session), challenge, nil
}

type session struct{}

func (session) Challenge(resp []byte) (bool, []byte, error) {
	if len(resp) == 0 {
		return false, nil, errors.New("empty challenge response: failed")
	}
	return true, nil, nil
}

const service = "kafka-cluster"

func challenge(auth Auth, host string) ([]byte, error) {
	host, _, err := net.SplitHostPort(host) // we do not need the port
	if err != nil {
		return nil, err
	}
	region, err := identifyRegion(host)
	if err != nil {
		return nil, err
	}

	var (
		timestamp = time.Now().UTC().Format("20060102T150405Z")
		date      = timestamp[:8] // 20060102
		scope     = scope(date, region)
		v         = make(url.Values)
	)

	v.Set("Action", service+":Connect")
	v.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	v.Set("X-Amz-Credential", auth.AccessKey+"/"+scope)
	v.Set("X-Amz-Date", timestamp)
	v.Set("X-Amz-Expires", "300") // 5 min
	v.Set("X-Amz-SignedHeaders", "host")
	if auth.SessionToken != "" {
		v.Set("X-Amz-Security-Token", auth.SessionToken)
	}

	qps := strings.ReplaceAll(v.Encode(), "+", "%20")

	canonicalRequest := task1(host, qps)
	sts := task2(timestamp, scope, canonicalRequest)
	signature := task3(auth.SecretKey, region, date, sts)

	v.Set("X-Amz-Signature", signature) // task4

	// According to the Java source and manual testing, all values in our
	// challenge map must be lowercased, and we MUST have host, and we MUST
	// have version, and version MUST be 2020_10_22.
	keyvals := make(map[string]string)
	for key, values := range v {
		keyvals[strings.ToLower(key)] = values[0]
	}
	keyvals["host"] = host
	keyvals["version"] = "2020_10_22"
	ua := auth.UserAgent
	if ua == "" {
		ua = strings.Join([]string{"franz-go", runtime.Version(), hostname}, "/")
	}
	keyvals["user-agent"] = ua

	marshaled, err := json.Marshal(keyvals)
	if err != nil {
		return nil, err
	}
	return marshaled, nil
}

// https://docs.aws.amazon.com/general/latest/gr/sigv4-create-string-to-sign.html
// "CredentialScope", Part 3
func scope(date, region string) string {
	return strings.Join([]string{date, region, service, "aws4_request"}, "/")
}

// https://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
func task1(host, qps string) []byte {
	// We start with our defined method, "GET", and the defined empty path,
	// "/". For query parameters, we have to escape +'s with %20, but we did
	// that above when building our URL.
	//
	//   HTTPRequestMethod + '\n' +
	//   CanonicalURI + '\n' +
	//   CanonicalQueryString + '\n' +
	canon := make([]byte, 0, 200)
	canon = append(canon, "GET\n"...)
	canon = append(canon, "/\n"...)
	canon = append(canon, qps...)
	canon = append(canon, '\n')

	// We only sign one header, the host. Each signed header is followed by
	// a newline, and then the canonical header block is followed itself by
	// a newline.
	//
	//   CanonicalHeaders + '\n' +
	//   SignedHeaders + '\n' +
	canon = append(canon, "host:"...)
	canon = append(canon, host...)
	canon = append(canon, "\n\nhost\n"...)

	// Finally, we add our empty body.
	//
	//   HexEncode(Hash(RequestPayload))
	const emptyBody = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	return append(canon, emptyBody...)
}

// https://docs.aws.amazon.com/general/latest/gr/sigv4-create-string-to-sign.html
func task2(timestamp, scope string, canonicalRequest []byte) []byte {
	toSign := make([]byte, 0, 512)
	toSign = append(toSign, "AWS4-HMAC-SHA256\n"...)
	toSign = append(toSign, timestamp...)
	toSign = append(toSign, '\n')
	toSign = append(toSign, scope...)
	toSign = append(toSign, '\n')
	canonHash := sha256.Sum256(canonicalRequest)
	hexBuf := make([]byte, 64) // 32 bytes to 64
	hex.Encode(hexBuf, canonHash[:])
	toSign = append(toSign, hexBuf...)
	return toSign
}

var aws4requestBytes = []byte("aws4_request")

// https://docs.aws.amazon.com/general/latest/gr/sigv4-calculate-signature.html
func task3(secretKey, region, date string, sts []byte) string {
	key := make([]byte, 0, 100)
	key = append(key, "AWS4"...)
	key = append(key, secretKey...)

	h := hmac.New(sha256.New, key)
	h.Write([]byte(date)) // kDate

	key = h.Sum(key[:0])
	h = hmac.New(sha256.New, key)
	h.Write([]byte(region)) // kRegion

	key = h.Sum(key[:0])
	h = hmac.New(sha256.New, key)
	h.Write([]byte(service)) // kService

	key = h.Sum(key[:0])
	h = hmac.New(sha256.New, key)
	h.Write(aws4requestBytes) // kSigning

	key = h.Sum(key[:0])
	h = hmac.New(sha256.New, key)
	h.Write(sts)

	return hex.EncodeToString(h.Sum(key[:0]))
}

// aws-java-sdk-core/src/main/resources/com/amazonaws/partitions/endpoints.json
var suffixes = []string{
	".amazonaws.com",
	".amazonaws.com.cn",
	".c2s.ic.gov",
	".sc2s.sgov.gov",
}

// aws-java-sdk-core/src/main/java/com/amazonaws/partitions/PartitionMetadataProvider.java
// tryGetRegionByEndpointDnsSuffix
func identifyRegion(host string) (string, error) {
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) {
			serviceRegion := strings.TrimSuffix(host, suffix)
			regionDot := strings.LastIndexByte(serviceRegion, '.')
			if regionDot == -1 {
				break
			}
			return serviceRegion[regionDot+1:], nil
		}
	}
	return "", fmt.Errorf("cannot determine the region in %+q", host)
}
