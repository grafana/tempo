package model

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	for _, e := range allEncodings {
		trace := test.MakeTrace(100, nil)
		bytes, err := marshal(trace, e)
		require.NoError(t, err)

		actual, err := Unmarshal(bytes, e)
		require.NoError(t, err)

		assert.True(t, proto.Equal(trace, actual))
	}
}
