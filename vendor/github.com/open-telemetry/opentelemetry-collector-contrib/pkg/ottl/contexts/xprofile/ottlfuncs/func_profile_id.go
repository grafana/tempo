// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/xprofile/ottlfuncs"

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	profileIDFuncName = "ProfileID"
	profileIDLen      = 16
	profileIDHexLen   = profileIDLen * 2
)

var (
	errDecodeProfileID    = errors.New("could not decode ID")
	errProfileIDLength    = fmt.Errorf("%w: %w", errDecodeProfileID, errors.New("invalid length"))
	errProfileIDHexDecode = fmt.Errorf("%w: %w", errDecodeProfileID, errors.New("invalid hex"))
)

type ProfileIDArguments[K any] struct {
	Target ottl.ByteSliceLikeGetter[K]
}

func NewProfileIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory(profileIDFuncName, &ProfileIDArguments[K]{}, createProfileIDFunction[K], ottl.WithExperimental[K]())
}

func createProfileIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ProfileIDArguments[K])
	if !ok {
		return nil, errors.New("ProfileIDFactory args must be of type *ProfileIDArguments[K]")
	}

	return profileID[K](args.Target)
}

func profileID[K any](target ottl.ByteSliceLikeGetter[K]) (ottl.ExprFunc[K], error) {
	// If the target is a literal getter, the ID is pre-computed once for optimal performance.
	if b, _, isLiteral := ottl.TryGetLiteralValue(target); isLiteral {
		result, err := bytesToProfileID(b)
		if err != nil {
			return nil, err
		}
		return func(context.Context, K) (any, error) {
			return result, nil
		}, nil
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		b, _, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		return bytesToProfileID(b)
	}, nil
}

// bytesToProfileID accepts either raw bytes of length profileIDLen or hex-encoded bytes of length profileIDHexLen.
func bytesToProfileID(b []byte) (any, error) {
	var id pprofile.ProfileID
	switch len(b) {
	case profileIDLen:
		copy(id[:], b)
		return id, nil
	case profileIDHexLen:
		if _, err := hex.Decode(id[:], b); err != nil {
			return nil, fmt.Errorf("%s: %w: %w", profileIDFuncName, errProfileIDHexDecode, err)
		}
		return id, nil
	default:
		return nil, fmt.Errorf("%s: %w: expected %d or %d bytes, got %d", profileIDFuncName, errProfileIDLength, profileIDLen, profileIDHexLen, len(b))
	}
}
