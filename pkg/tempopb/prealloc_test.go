package tempopb

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	var dummyData = make([]byte, 10)
	rand.Read(dummyData)

	preallocReq := &PreallocBytes{}
	err := preallocReq.Unmarshal(dummyData)
	assert.NoError(t, err)

	assert.Equal(t, dummyData, preallocReq.Slice)
}

func TestMarshal(t *testing.T) {
	preallocReq := &PreallocBytes{
		Slice: make([]byte, 10),
	}
	rand.Read(preallocReq.Slice)

	var dummyData = make([]byte, 10)
	_, err := preallocReq.MarshalTo(dummyData)
	assert.NoError(t, err)

	assert.Equal(t, preallocReq.Slice, dummyData)
}

func TestSize(t *testing.T) {
	preallocReq := &PreallocBytes{
		Slice: make([]byte, 10),
	}
	assert.Equal(t, 10, preallocReq.Size())
}
