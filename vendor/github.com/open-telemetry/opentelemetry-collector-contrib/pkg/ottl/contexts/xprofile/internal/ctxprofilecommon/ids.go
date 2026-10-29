// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxprofilecommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/xprofile/internal/ctxprofilecommon"

import (
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

func ParseProfileID(profileIDStr string) (pprofile.ProfileID, error) {
	var id pprofile.ProfileID
	if hex.DecodedLen(len(profileIDStr)) != len(id) {
		return pprofile.ProfileID{}, errors.New("profile ids must be 32 hex characters")
	}
	_, err := hex.Decode(id[:], []byte(profileIDStr))
	if err != nil {
		return pprofile.ProfileID{}, err
	}
	return id, nil
}
