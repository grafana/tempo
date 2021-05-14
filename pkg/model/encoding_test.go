package model

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	empty := &tempopb.Trace{}

	for _, e := range allEncodings {
		trace := test.MakeTrace(100, nil)
		bytes, err := marshal(trace, e)
		require.NoError(t, err)

		actual, err := Unmarshal(bytes, e)
		require.NoError(t, err)
		assert.True(t, proto.Equal(trace, actual))

		actual, err = Unmarshal(nil, e)
		assert.NoError(t, err)
		assert.True(t, proto.Equal(empty, actual))

		actual, err = Unmarshal([]byte{}, e)
		assert.NoError(t, err)
		assert.True(t, proto.Equal(empty, actual))
	}
}
