// This is copied over from Jaeger and modified to work for Tempo

package gogocodec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/grafana/tempo/pkg/tempopb"
)

type mockOTELProtoMessage struct {
	Val byte
}

func (m *mockOTELProtoMessage) SizeProto() int {
	return 1
}

func (m *mockOTELProtoMessage) MarshalProto(buf []byte) int {
	buf[0] = m.Val
	return 1
}

func (m *mockOTELProtoMessage) UnmarshalProto(buf []byte) error {
	m.Val = buf[0]
	return nil
}

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
	assert.True(t, proto.Equal(goprotoMessage1, goprotoMessage2))
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

func TestCodecMarshallAndUnmarshall_otel_proto_type(t *testing.T) {
	c := NewCodec()

	req1 := &mockOTELProtoMessage{Val: 42}
	data, err := c.Marshal(req1)
	require.NoError(t, err)

	req2 := &mockOTELProtoMessage{}
	err = c.Unmarshal(data, req2)
	require.NoError(t, err)
	assert.Equal(t, req1, req2)
}

func TestCodecMarshal_unsupported_type(t *testing.T) {
	c := NewCodec()

	_, err := c.Marshal(struct{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported marshal type")
}

func TestCodecUnmarshal_unsupported_type(t *testing.T) {
	c := NewCodec()

	err := c.Unmarshal([]byte{0x01}, struct{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported unmarshal type")
}
