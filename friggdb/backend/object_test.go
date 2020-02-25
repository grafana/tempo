package backend

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshal(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := []byte{0x00, 0x01}
	req := test.MakeRequest(10, id)

	bReq, err := proto.Marshal(req)
	assert.NoError(t, err)

	_, err = MarshalObjectToWriter(id, bReq, buffer)
	assert.NoError(t, err)

	outID, outObject, err := unmarshalObjectFromReader(buffer)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(id, outID))

	outReq := &friggpb.PushRequest{}
	err = proto.Unmarshal(outObject, outReq)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(req, outReq))
}

func TestMarshalUnmarshalFromBuffer(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := []byte{0x00, 0x01}

	var reqs []*friggpb.PushRequest
	for i := 0; i < 10; i++ {
		req := test.MakeRequest(10, id)
		reqs = append(reqs, req)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)

		_, err = MarshalObjectToWriter(id, bReq, buffer)
		assert.NoError(t, err)
	}

	actualBuffer := buffer.Bytes()
	for i := 0; i < 10; i++ {
		var outID []byte
		var outObject []byte
		var err error
		actualBuffer, outID, outObject, err = unmarshalAndAdvanceBuffer(actualBuffer)
		assert.NoError(t, err)

		outReq := &friggpb.PushRequest{}
		err = proto.Unmarshal(outObject, outReq)
		assert.NoError(t, err)

		assert.True(t, bytes.Equal(id, outID))
		assert.True(t, proto.Equal(reqs[i], outReq))
	}
}
