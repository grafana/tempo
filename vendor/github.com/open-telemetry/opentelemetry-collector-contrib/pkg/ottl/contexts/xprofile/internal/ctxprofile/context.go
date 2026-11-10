// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxprofile // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/xprofile/internal/ctxprofile"
import (
	"go.opentelemetry.io/collector/pdata/pprofile"
)

const (
	Name   = "profile"
	DocRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/xprofile/ottlprofile"
)

type Context interface {
	GetProfile() pprofile.Profile
	GetProfilesDictionary() pprofile.ProfilesDictionary
}
