// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configauth

import (
	"context"

	"google.golang.org/grpc"
)

// Authentication defines the auth settings for the receiver
type Authentication struct {
	// The attribute (header name) to look for auth data. Optional, default value: "authentication".
	Attribute string `mapstructure:"attribute"`

	// OIDC configures this receiver to use the given OIDC provider as the backend for the authentication mechanism.
	// Required.
	OIDC *OIDC `mapstructure:"oidc"`
}

// OIDC defines the OpenID Connect properties for this processor
type OIDC struct {
	// IssuerURL is the base URL for the OIDC provider.
	// Required.
	IssuerURL string `mapstructure:"issuer_url"`

	// Audience of the token, used during the verification.
	// For example: "https://accounts.google.com" or "https://login.salesforce.com".
	// Required.
	Audience string `mapstructure:"audience"`

	// The local path for the issuer CA's TLS server cert.
	// Optional.
	IssuerCAPath string `mapstructure:"issuer_ca_path"`

	// The claim to use as the username, in case the token's 'sub' isn't the suitable source.
	// Optional.
	UsernameClaim string `mapstructure:"username_claim"`

	// The claim that holds the subject's group membership information.
	// Optional.
	GroupsClaim string `mapstructure:"groups_claim"`
}

// ToServerOptions builds a set of server options ready to be used by the gRPC server
func (a *Authentication) ToServerOptions() ([]grpc.ServerOption, error) {
	auth, err := NewAuthenticator(*a)
	if err != nil {
		return nil, err
	}

	// perhaps we should use a timeout here?
	// TODO: we need a hook to call auth.Close()
	if err := auth.Start(context.Background()); err != nil {
		return nil, err
	}

	return []grpc.ServerOption{
		grpc.UnaryInterceptor(auth.UnaryInterceptor),
		grpc.StreamInterceptor(auth.StreamInterceptor),
	}, nil
}
