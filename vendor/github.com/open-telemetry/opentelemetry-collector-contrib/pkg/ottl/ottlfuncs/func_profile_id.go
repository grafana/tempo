// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const profileIDFuncName = "ProfileID"

type ProfileIDArguments[K any] struct {
	Target ottl.ByteSliceLikeGetter[K]
}

func NewProfileIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory(profileIDFuncName, &ProfileIDArguments[K]{}, createProfileIDFunction[K])
}

func createProfileIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ProfileIDArguments[K])

	if !ok {
		return nil, errors.New("ProfileIDFactory args must be of type *ProfileIDArguments[K]")
	}

	return profileID[K](args.Target)
}

func profileID[K any](target ottl.ByteSliceLikeGetter[K]) (ottl.ExprFunc[K], error) {
	return newIDExprFunc(profileIDFuncName, target, decodeHexToProfileID)
}

func decodeHexToProfileID(b []byte) (pprofile.ProfileID, error) {
	var id pprofile.ProfileID
	if _, err := hex.Decode(id[:], b); err != nil {
		return pprofile.ProfileID{}, err
	}
	return id, nil
}
