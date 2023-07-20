package tempopb

import (
	crand "crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	dummyData := make([]byte, 10)
	_, err := crand.Read(dummyData)
	require.NoError(t, err)

	preallocReq := &PreallocBytes{}
	err = preallocReq.Unmarshal(dummyData)
	assert.NoError(t, err)

	assert.Equal(t, dummyData, preallocReq.Slice)
}

func TestMarshal(t *testing.T) {
	preallocReq := &PreallocBytes{
		Slice: make([]byte, 10),
	}
	_, err := crand.Read(preallocReq.Slice)
	require.NoError(t, err)

	dummyData := make([]byte, 10)
	_, err = preallocReq.MarshalTo(dummyData)
	assert.NoError(t, err)

	assert.Equal(t, preallocReq.Slice, dummyData)
}

func TestSize(t *testing.T) {
	preallocReq := &PreallocBytes{
		Slice: make([]byte, 10),
	}
	assert.Equal(t, 10, preallocReq.Size())
}
