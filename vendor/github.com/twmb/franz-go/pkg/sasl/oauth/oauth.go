// Package oauth provides OAUTHBEARER sasl authentication as specified in
// RFC7628.
package oauth

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/twmb/franz-go/pkg/sasl"
)

// Auth contains information for authentication.
//
// This client may add fields to this struct in the future if Kafka adds more
// capabilities to Oauth.
type Auth struct {
	// Zid is an optional authorization ID to use in authenticating.
	Zid string

	// Token is the oauthbearer token to use for a single session's
	// authentication.
	Token string
	// Extensions are key value pairs to add to the authentication request.
	Extensions map[string]string

	_ struct{} // require explicit field initialization
}

// AsMechanism returns a sasl mechanism that will use 'a' as credentials for
// all sasl sessions.
//
// This is a shortcut for using the Oauth function and is useful when you do
// not need to live-rotate credentials.
func (a Auth) AsMechanism() sasl.Mechanism {
	return Oauth(func(context.Context) (Auth, error) {
		return a, nil
	})
}

// Oauth returns an OAUTHBEARER sasl mechanism that will call authFn whenever
// authentication is needed. The returned Auth is used for a single session.
//
// authFn is called for every new connection and every reauthentication; it is
// recommended to cache the token and fetch a new one only when the cached
// token is close to expiring.
func Oauth(authFn func(context.Context) (Auth, error)) sasl.Mechanism {
	return oauth(authFn)
}

type oauth func(context.Context) (Auth, error)

func (oauth) Name() string { return "OAUTHBEARER" }
func (fn oauth) Authenticate(ctx context.Context, _ string) (sasl.Session, []byte, error) {
	auth, err := fn(ctx)
	if err != nil {
		return nil, nil, err
	}
	if auth.Token == "" {
		return nil, nil, errors.New("OAUTHBEARER token must be non-empty")
	}

	// https://tools.ietf.org/html/rfc7628#section-3.1
	gs2Header := "n," // no channel binding
	if auth.Zid != "" {
		gs2Header += "a=" + auth.Zid
	}
	gs2Header += ","
	init := []byte(gs2Header + "\x01auth=Bearer ")
	init = append(init, auth.Token...)
	init = append(init, '\x01')
	// We sort extensions for consistency, but it is not required.
	for _, k := range slices.Sorted(maps.Keys(auth.Extensions)) {
		if k == "" {
			continue
		}
		init = append(init, k...)
		init = append(init, '=')
		init = append(init, auth.Extensions[k]...)
		init = append(init, '\x01')
	}
	init = append(init, '\x01')

	return &session{}, init, nil
}

type session struct {
	serverErr string
}

func (s *session) Challenge(resp []byte) (bool, []byte, error) {
	// A non-empty challenge is the server's error (RFC 7628 section
	// 3.2.2). We reply with a single %x01, and Kafka then fails the
	// authentication with SASL_AUTHENTICATION_FAILED and the error as
	// the message. We are only called again if the server did not fail.
	if s.serverErr != "" {
		return false, nil, fmt.Errorf("OAUTHBEARER authentication failed: %s", s.serverErr)
	}
	if len(resp) != 0 {
		s.serverErr = string(resp)
		return false, []byte{'\x01'}, nil
	}
	return true, nil, nil
}
