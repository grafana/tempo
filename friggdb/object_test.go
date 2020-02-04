package friggdb

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

	_, err := marshalObjectToWriter(id, req, buffer)
	assert.NoError(t, err)

	outReq := &friggpb.PushRequest{}
	outID, found, err := unmarshalObjectFromReader(outReq, buffer)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, bytes.Equal(id, outID))
	assert.True(t, proto.Equal(req, outReq))
}
