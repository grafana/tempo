// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type idExprBuilder func(ottl.ByteSliceLikeGetter[any]) (ottl.ExprFunc[any], error)

type idSuccessTestCase struct {
	name  string
	value []byte
	want  any
}

type idErrorTestCase struct {
	name  string
	value []byte
	err   error
}

// makeIDGetter creates a ByteSliceLikeGetter for testing purposes.
// This is a shared helper used by TraceID, SpanID, and ProfileID tests.
func makeIDGetter(bytes []byte) ottl.ByteSliceLikeGetter[any] {
	return ottl.StandardByteSliceLikeGetter[any]{Getter: func(_ context.Context, _ any) (any, error) {
		return bytes, nil
	}}
}

func runIDSuccessTests(t *testing.T, builder idExprBuilder, cases []idSuccessTestCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := builder(makeIDGetter(tt.value))
			require.NoError(t, err)
			result, err := expr(t.Context(), nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func runIDErrorTests(t *testing.T, builder idExprBuilder, funcName string, cases []idErrorTestCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Dynamic getters succeed at init, fail at execution
			expr, err := builder(makeIDGetter(tt.value))
			require.NoError(t, err, "initialization should succeed for dynamic getters")

			result, err := expr(t.Context(), nil)

			assert.Nil(t, result)
			assertErrorIsFuncDecode(t, err, funcName)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

// assertErrorIsFuncDecode asserts that the error message contains the function name.
func assertErrorIsFuncDecode(t *testing.T, err error, funcName string) {
	t.Helper()
	assert.ErrorIs(t, err, errDecodeID)
	assert.ErrorContains(t, err, funcName)
}
