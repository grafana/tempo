// Package oauth provides OAUTHBEARER sasl authentication as specified in
// RFC7628.
package oauth

import (
	"context"
	"errors"
	"sort"

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

	// We sort extensions for consistency, but it is not required.
	type kv struct {
		k string
		v string
	}
	kvs := make([]kv, 0, len(auth.Extensions))
	for k, v := range auth.Extensions {
		if len(k) == 0 {
			continue
		}
		kvs = append(kvs, kv{k, v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })

	// https://tools.ietf.org/html/rfc7628#section-3.1
	gs2Header := "n," // no channel binding
	if auth.Zid != "" {
		gs2Header += "a=" + auth.Zid
	}
	gs2Header += ","
	init := []byte(gs2Header + "\x01auth=Bearer ")
	init = append(init, auth.Token...)
	init = append(init, '\x01')
	for _, kv := range kvs {
		init = append(init, kv.k...)
		init = append(init, '=')
		init = append(init, kv.v...)
		init = append(init, '\x01')
	}
	init = append(init, '\x01')

	return session{}, init, nil
}

type session struct{}

func (session) Challenge(resp []byte) (bool, []byte, error) {
	if len(resp) != 0 {
		return false, nil, errors.New("unexpected data in oauth response")
	}
	return true, nil, nil
}
