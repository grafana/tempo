package model

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	empty := &tempopb.Trace{}

	for _, e := range allEncodings {
		encoding, err := NewEncoding(e)
		require.NoError(t, err)

		// random trace
		trace := test.MakeTrace(100, nil)
		bytes, err := encoding.Marshal(trace)
		require.NoError(t, err)

		actual, err := encoding.Unmarshal(bytes)
		require.NoError(t, err)
		assert.True(t, proto.Equal(trace, actual))

		// nil trace
		actual, err = encoding.Unmarshal(nil)
		assert.NoError(t, err)
		assert.True(t, proto.Equal(empty, actual))

		// empty byte slice
		actual, err = encoding.Unmarshal([]byte{})
		assert.NoError(t, err)
		assert.True(t, proto.Equal(empty, actual))
	}
}
