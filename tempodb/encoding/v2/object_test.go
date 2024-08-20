package v2

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/golang/protobuf/proto" //nolint:all //ProtoReflect
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/grafana/tempo/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshal(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := []byte{0x00, 0x01}
	req := test.MakeTrace(10, id)

	bReq, err := proto.Marshal(req)
	assert.NoError(t, err)

	o := object{}
	_, err = o.MarshalObjectToWriter(id, bReq, buffer)
	assert.NoError(t, err)

	outID, outObject, err := o.UnmarshalObjectFromReader(buffer)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(id, outID))

	outReq := &tempopb.Trace{}
	err = proto.Unmarshal(outObject, outReq)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(req, outReq))
}

func TestMarshalUnmarshalFromBuffer(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := make([]byte, 16)
	_, err := rand.Read(id)
	assert.NoError(t, err)

	o := object{}
	var reqs []*tempopb.Trace
	for i := 0; i < 10; i++ {
		req := test.MakeTrace(10, id)
		reqs = append(reqs, req)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)

		_, err = o.MarshalObjectToWriter(id, bReq, buffer)
		assert.NoError(t, err)
	}

	actualBuffer := buffer.Bytes()
	for i := 0; i < 10; i++ {
		var outID []byte
		var outObject []byte
		var err error
		actualBuffer, outID, outObject, err = o.UnmarshalAndAdvanceBuffer(actualBuffer)
		assert.NoError(t, err)

		outReq := &tempopb.Trace{}
		err = proto.Unmarshal(outObject, outReq)
		assert.NoError(t, err)

		assert.True(t, bytes.Equal(id, outID))
		assert.True(t, proto.Equal(reqs[i], outReq))
	}
}
