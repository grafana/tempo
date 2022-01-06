package model

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) { // jpe expand
	empty := &tempopb.Trace{}

	for _, e := range allEncodings {
		decoder, err := NewDecoder(e)
		require.NoError(t, err)

		trace := test.MakeTrace(100, nil)
		bytes, err := decoder.Marshal(trace)
		require.NoError(t, err)

		actual, err := decoder.Unmarshal(bytes)
		require.NoError(t, err)
		assert.True(t, proto.Equal(trace, actual))

		actual, err = decoder.Unmarshal(nil)
		assert.NoError(t, err)
		assert.True(t, proto.Equal(empty, actual))

		actual, err = decoder.Unmarshal([]byte{})
		assert.NoError(t, err)
		assert.True(t, proto.Equal(empty, actual))
	}
}
