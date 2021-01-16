package v0

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := []byte{0x00, 0x01}
	req := test.MakeRequest(10, id)

	bReq, err := proto.Marshal(req)
	assert.NoError(t, err)

	_, err = marshalObjectToWriter(id, bReq, buffer)
	assert.NoError(t, err)

	outID, outObject, err := unmarshalObjectFromReader(buffer)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(id, outID))

	outReq := &tempopb.PushRequest{}
	err = proto.Unmarshal(outObject, outReq)
	assert.NoError(t, err)

	assert.True(t, proto.Equal(req, outReq))
}

func TestMarshalUnmarshalFromBuffer(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01}
	rand.Read(id)

	var reqs []*tempopb.PushRequest
	for i := 0; i < 10; i++ {
		req := test.MakeRequest(10, id)
		reqs = append(reqs, req)

		bReq, err := proto.Marshal(req)
		assert.NoError(t, err)

		_, err = marshalObjectToWriter(id, bReq, buffer)
		assert.NoError(t, err)
	}

	actualBuffer := buffer.Bytes()
	for i := 0; i < 10; i++ {
		var outID []byte
		var outObject []byte
		var err error
		actualBuffer, outID, outObject, err = unmarshalAndAdvanceBuffer(actualBuffer)
		assert.NoError(t, err)

		outReq := &tempopb.PushRequest{}
		err = proto.Unmarshal(outObject, outReq)
		assert.NoError(t, err)

		assert.True(t, bytes.Equal(id, outID))
		assert.True(t, proto.Equal(reqs[i], outReq))
	}
}

func TestMarshalUnmarshalRecordFromBuffer(t *testing.T) {
	buffer := &bytes.Buffer{}
	id := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01}
	rand.Read(id)

	var reqs []*tempopb.PushRequest
	for i := 0; i < 10; i++ {
		req := test.MakeRequest(10, id)
		reqs = append(reqs, req)

		bReq, err := proto.Marshal(req)
		require.NoError(t, err)

		_, err = marshalObjectToWriter(id, bReq, buffer)
		require.NoError(t, err)
	}

	bCopy := make([]byte, buffer.Len())
	copy(bCopy, buffer.Bytes())

	records := []*Record{}
	var offset uint64
	for i := 0; i < 10; i++ {
		record, err := unmarshalObjectRecordFromReader(buffer, &offset)
		require.NoError(t, err)
		records = append(records, record)
	}

	record, err := unmarshalObjectRecordFromReader(buffer, &offset)
	require.Nil(t, err)
	require.Nil(t, record)

	readBuffer := bytes.NewReader(bCopy)
	for i := 0; i < 10; i++ {
		record := records[i]
		objBuffer := make([]byte, int(record.Length))

		_, err := readBuffer.ReadAt(objBuffer, int64(record.Start))
		require.NoError(t, err)

		id, actualObject, err := unmarshalObjectFromReader(bytes.NewBuffer(objBuffer))
		require.NoError(t, err)

		// id equal
		assert.Equal(t, record.ID, id)

		outReq := &tempopb.PushRequest{}
		err = proto.Unmarshal(actualObject, outReq)
		assert.NoError(t, err)

		// object equal
		assert.True(t, proto.Equal(reqs[i], outReq))
	}
}
