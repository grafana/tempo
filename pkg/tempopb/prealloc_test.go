package tempopb

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	var dummyData = make([]byte, 10)
	rand.Read(dummyData)

	preallocReq := &PreallocRequest{}
	err := preallocReq.Unmarshal(dummyData)
	assert.NoError(t, err)

	assert.Equal(t, dummyData, preallocReq.Request)
}

func TestMarshal(t *testing.T) {
	preallocReq := &PreallocRequest{
		Request: make([]byte, 10),
	}
	rand.Read(preallocReq.Request)

	var dummyData = make([]byte, 10)
	_, err := preallocReq.MarshalTo(dummyData)
	assert.NoError(t, err)

	assert.Equal(t, preallocReq.Request, dummyData)
}

func TestSize(t *testing.T) {
	preallocReq := &PreallocRequest{
		Request: make([]byte, 10),
	}
	assert.Equal(t, 10, preallocReq.Size())
}

func TestReuseRequest(t *testing.T) {
	tests := []struct {
		name          string
		donate        int
		request       int
		expectedEqual bool
	}{
		{
			name:          "same size",
			donate:        1500,
			request:       1500,
			expectedEqual: true,
		},
		{
			name:          "larger donate - same bucket",
			donate:        1600,
			request:       1500,
			expectedEqual: true,
		},
		{
			name:    "larger donate - different bucket",
			donate:  2100,
			request: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create push requests of known size
			req := MakeBytesRequestWithSize(tt.donate)
			assert.Len(t, req.Requests, 1)
			expectedAddr := &req.Requests[0].Request[0]

			// "donate" to bytePool
			ReuseRequest(req)

			// unmarshal a new request
			var dummyData = make([]byte, tt.request)
			preallocReq := &PreallocRequest{}
			assert.NoError(t, preallocReq.Unmarshal(dummyData))
			actualAddr := &preallocReq.Request[0]

			if tt.expectedEqual {
				assert.Equal(t, expectedAddr, actualAddr)
			} else {
				assert.NotEqual(t, expectedAddr, actualAddr)
			}
		})
	}
}

func MakeBytesRequestWithSize(maxBytes int) *PushBytesRequest {
	reqBytes := make([]byte, maxBytes)
	rand.Read(reqBytes)

	return &PushBytesRequest{
		Requests: []PreallocRequest{
			{
				Request: reqBytes,
			},
		},
	}
}
