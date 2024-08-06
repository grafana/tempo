// This is copied over from Jaeger and modified to work for Tempo

package gogocodec

import (
	"testing"

	"github.com/golang/protobuf/proto" //nolint:all,deprecated SA1019 deprecated package
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/grafana/tempo/v2/pkg/tempopb"
)

func TestCodecMarshallAndUnmarshall_tempo_type(t *testing.T) {
	// marshal a tempo object using the custom codec
	c := NewCodec()
	req1 := &tempopb.TraceByIDRequest{
		TraceID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	data, err := c.Marshal(req1)
	require.NoError(t, err)

	// unmarshal and check if its the same
	req2 := &tempopb.TraceByIDRequest{}
	err = c.Unmarshal(data, req2)
	require.NoError(t, err)
	assert.Equal(t, req1, req2)
}

func TestCodecMarshallAndUnmarshall_foreign_type(t *testing.T) {
	// marshal a foreign object (anything other than Tempo/Cortex/Jaeger) using the custom codec
	c := NewCodec()
	goprotoMessage1 := &emptypb.Empty{}
	data, err := c.Marshal(goprotoMessage1)
	require.NoError(t, err)

	// unmarshal and check if its the same
	goprotoMessage2 := &emptypb.Empty{}
	err = c.Unmarshal(data, goprotoMessage2)
	require.NoError(t, err)
	assert.Equal(t, goprotoMessage1, goprotoMessage2)
}

func TestWireCompatibility(t *testing.T) {
	// marshal a tempo object using the custom codec
	c := NewCodec()
	req1 := &tempopb.TraceByIDRequest{
		TraceID: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
	}
	data, err := c.Marshal(req1)
	require.NoError(t, err)

	// unmarshal this into the generic empty type using golang proto
	var goprotoMessage emptypb.Empty
	err = proto.Unmarshal(data, &goprotoMessage)
	require.NoError(t, err)

	// marshal emptypb using golang proto
	data2, err := proto.Marshal(&goprotoMessage)
	require.NoError(t, err)

	req2 := &tempopb.TraceByIDRequest{}
	err = c.Unmarshal(data2, req2)
	require.NoError(t, err)
	assert.Equal(t, req1, req2)
}
