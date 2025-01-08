// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package impersonate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
	"cloud.google.com/go/auth/internal"
)

var (
	universeDomainPlaceholder                   = "UNIVERSE_DOMAIN"
	iamCredentialsEndpoint                      = "https://iamcredentials.googleapis.com"
	iamCredentialsUniverseDomainEndpoint        = "https://iamcredentials.UNIVERSE_DOMAIN"
	oauth2Endpoint                              = "https://oauth2.googleapis.com"
	errMissingTargetPrincipal                   = errors.New("impersonate: target service account must be provided")
	errMissingScopes                            = errors.New("impersonate: scopes must be provided")
	errLifetimeOverMax                          = errors.New("impersonate: max lifetime is 12 hours")
	errUniverseNotSupportedDomainWideDelegation = errors.New("impersonate: service account user is configured for the credential. " +
		"Domain-wide delegation is not supported in universes other than googleapis.com")
)

// TODO(codyoss): plumb through base for this and idtoken

// NewCredentials returns an impersonated
// [cloud.google.com/go/auth/NewCredentials] configured with the provided options
// and using credentials loaded from Application Default Credentials as the base
// credentials if not provided with the opts.
func NewCredentials(opts *CredentialsOptions) (*auth.Credentials, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	var isStaticToken bool
	// Default to the longest acceptable value of one hour as the token will
	// be refreshed automatically if not set.
	lifetime := 1 * time.Hour
	if opts.Lifetime != 0 {
		lifetime = opts.Lifetime
		// Don't auto-refresh token if a lifetime is configured.
		isStaticToken = true
	}

	client := opts.Client
	creds := opts.Credentials
	if client == nil {
		var err error
		if creds == nil {
			creds, err = credentials.DetectDefault(&credentials.DetectOptions{
				Scopes:           []string{defaultScope},
				UseSelfSignedJWT: true,
			})
			if err != nil {
				return nil, err
			}
		}

		client, err = httptransport.NewClient(transportOpts(opts, creds))
		if err != nil {
			return nil, err
		}
	}

	universeDomainProvider := resolveUniverseDomainProvider(creds)
	// If a subject is specified a domain-wide delegation auth-flow is initiated
	// to impersonate as the provided subject (user).
	if opts.Subject != "" {
		tp, err := user(opts, client, lifetime, isStaticToken, universeDomainProvider)
		if err != nil {
			return nil, err
		}
		return auth.NewCredentials(&auth.CredentialsOptions{
			TokenProvider:          tp,
			UniverseDomainProvider: universeDomainProvider,
		}), nil
	}

	its := impersonatedTokenProvider{
		client:                 client,
		targetPrincipal:        opts.TargetPrincipal,
		lifetime:               fmt.Sprintf("%.fs", lifetime.Seconds()),
		universeDomainProvider: universeDomainProvider,
	}
	for _, v := range opts.Delegates {
		its.delegates = append(its.delegates, formatIAMServiceAccountName(v))
	}
	its.scopes = make([]string, len(opts.Scopes))
	copy(its.scopes, opts.Scopes)

	var tpo *auth.CachedTokenProviderOptions
	if isStaticToken {
		tpo = &auth.CachedTokenProviderOptions{
			DisableAutoRefresh: true,
		}
	}

	return auth.NewCredentials(&auth.CredentialsOptions{
		TokenProvider:          auth.NewCachedTokenProvider(its, tpo),
		UniverseDomainProvider: universeDomainProvider,
	}), nil
}

// transportOpts returns options for httptransport.NewClient. If opts.UniverseDomain
// is provided, it will be used in the transport for a validation ensuring that it
// matches the universe domain in the base credentials. If opts.UniverseDomain
// is not provided, this validation will be skipped.
func transportOpts(opts *CredentialsOptions, creds *auth.Credentials) *httptransport.Options {
	tOpts := &httptransport.Options{
		Credentials: creds,
	}
	if opts.UniverseDomain == "" {
		tOpts.InternalOptions = &httptransport.InternalOptions{
			SkipUniverseDomainValidation: true,
		}
	} else {
		tOpts.UniverseDomain = opts.UniverseDomain
	}
	return tOpts
}

// resolveUniverseDomainProvider returns the default service domain for a given
// Cloud universe. This is the universe domain configured for the credentials,
// which will be used in endpoint(s), and compared to the universe domain that
// is separately configured for the client.
func resolveUniverseDomainProvider(creds *auth.Credentials) auth.CredentialsPropertyProvider {
	if creds != nil {
		return auth.CredentialsPropertyFunc(creds.UniverseDomain)
	}
	return internal.StaticCredentialsProperty(internal.DefaultUniverseDomain)
}

// CredentialsOptions for generating an impersonated credential token.
type CredentialsOptions struct {
	// TargetPrincipal is the email address of the service account to
	// impersonate. Required.
	TargetPrincipal string
	// Scopes that the impersonated credential should have. Required.
	Scopes []string
	// Delegates are the service account email addresses in a delegation chain.
	// Each service account must be granted roles/iam.serviceAccountTokenCreator
	// on the next service account in the chain. Optional.
	Delegates []string
	// Lifetime is the amount of time until the impersonated token expires. If
	// unset the token's lifetime will be one hour and be automatically
	// refreshed. If set the token may have a max lifetime of one hour and will
	// not be refreshed. Service accounts that have been added to an org policy
	// with constraints/iam.allowServiceAccountCredentialLifetimeExtension may
	// request a token lifetime of up to 12 hours. Optional.
	Lifetime time.Duration
	// Subject is the sub field of a JWT. This field should only be set if you
	// wish to impersonate as a user. This feature is useful when using domain
	// wide delegation. Optional.
	Subject string

	// Credentials used in generating the impersonated token. If empty, an
	// attempt will be made to detect credentials from the environment (see
	// [cloud.google.com/go/auth/credentials.DetectDefault]). Optional.
	Credentials *auth.Credentials
	// Client configures the underlying client used to make network requests
	// when fetching tokens. If provided this should be a fully-authenticated
	// client. Optional.
	Client *http.Client
	// UniverseDomain is the default service domain for a given Cloud universe.
	// This field has no default value, and only if provided will it be used to
	// verify the universe domain from the credentials. Optional.
	UniverseDomain string
}

func (o *CredentialsOptions) validate() error {
	if o == nil {
		return errors.New("impersonate: options must be provided")
	}
	if o.TargetPrincipal == "" {
		return errMissingTargetPrincipal
	}
	if len(o.Scopes) == 0 {
		return errMissingScopes
	}
	if o.Lifetime.Hours() > 12 {
		return errLifetimeOverMax
	}
	return nil
}

func formatIAMServiceAccountName(name string) string {
	return fmt.Sprintf("projects/-/serviceAccounts/%s", name)
}

type generateAccessTokenRequest struct {
	Delegates []string `json:"delegates,omitempty"`
	Lifetime  string   `json:"lifetime,omitempty"`
	Scope     []string `json:"scope,omitempty"`
}

type generateAccessTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireTime  string `json:"expireTime"`
}

type impersonatedTokenProvider struct {
	client                 *http.Client
	universeDomainProvider auth.CredentialsPropertyProvider

	targetPrincipal string
	lifetime        string
	scopes          []string
	delegates       []string
}

// Token returns an impersonated Token.
func (i impersonatedTokenProvider) Token(ctx context.Context) (*auth.Token, error) {
	reqBody := generateAccessTokenRequest{
		Delegates: i.delegates,
		Lifetime:  i.lifetime,
		Scope:     i.scopes,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("impersonate: unable to marshal request: %w", err)
	}
	universeDomain, err := i.universeDomainProvider.GetProperty(ctx)
	if err != nil {
		return nil, err
	}
	endpoint := strings.Replace(iamCredentialsUniverseDomainEndpoint, universeDomainPlaceholder, universeDomain, 1)
	url := fmt.Sprintf("%s/v1/%s:generateAccessToken", endpoint, formatIAMServiceAccountName(i.targetPrincipal))
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("impersonate: unable to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, body, err := internal.DoRequest(i.client, req)
	if err != nil {
		return nil, fmt.Errorf("impersonate: unable to generate access token: %w", err)
	}
	if c := resp.StatusCode; c < 200 || c > 299 {
		return nil, fmt.Errorf("impersonate: status code %d: %s", c, body)
	}

	var accessTokenResp generateAccessTokenResponse
	if err := json.Unmarshal(body, &accessTokenResp); err != nil {
		return nil, fmt.Errorf("impersonate: unable to parse response: %w", err)
	}
	expiry, err := time.Parse(time.RFC3339, accessTokenResp.ExpireTime)
	if err != nil {
		return nil, fmt.Errorf("impersonate: unable to parse expiry: %w", err)
	}
	return &auth.Token{
		Value:  accessTokenResp.AccessToken,
		Expiry: expiry,
	}, nil
}
