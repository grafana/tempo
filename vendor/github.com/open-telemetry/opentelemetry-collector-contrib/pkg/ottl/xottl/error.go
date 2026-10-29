// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"

import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlerror"

// Error represents an error that occurred during parsing/evaluation of OTTL statements, conditions,
// and value expressions. It provides a message and optionally a position in the input where
// the error occurred.
type Error = ottlerror.Error
