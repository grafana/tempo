// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"fmt"
	"strings"
)

type ErrorMode string

const (
	IgnoreError    ErrorMode = "ignore"
	PropagateError ErrorMode = "propagate"
	SilentError    ErrorMode = "silent"
)

func (e *ErrorMode) UnmarshalText(text []byte) error {
	str := ErrorMode(strings.ToLower(string(text)))
	switch str {
	case IgnoreError, PropagateError, SilentError:
		*e = str
		return nil
	default:
		return fmt.Errorf("unknown error mode %v", str)
	}
}

type LogicOperation string

const (
	And LogicOperation = "and"
	Or  LogicOperation = "or"
)

func (l *LogicOperation) UnmarshalText(text []byte) error {
	str := LogicOperation(strings.ToLower(string(text)))
	switch str {
	case And, Or:
		*l = str
		return nil
	default:
		return fmt.Errorf("unknown LogicOperation %v", str)
	}
}
