package io

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testBufLength = 10

func TestReadAllWithEstimate(t *testing.T) {
	buf := make([]byte, testBufLength)
	_, err := rand.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, testBufLength, len(buf))
	assert.Equal(t, testBufLength, cap(buf))

	actualBuf, err := ReadAllWithEstimate(bytes.NewReader(buf), int64(testBufLength))
	assert.NoError(t, err)
	assert.Equal(t, buf, actualBuf)
	assert.Equal(t, testBufLength, len(actualBuf))
	assert.Equal(t, testBufLength+1, cap(actualBuf)) // one extra byte used in ReadAllWithEstimate
}
