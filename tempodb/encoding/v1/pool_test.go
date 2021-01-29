package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPool(t *testing.T) {
	for _, enc := range supportedEncoding {
		rPool, err := getReaderPool(enc)
		assert.NotNil(t, rPool)
		assert.NoError(t, err)

		wPool, err := getWriterPool(enc)
		assert.NotNil(t, wPool)
		assert.NoError(t, err)
	}

	rPool, err := getReaderPool(maxEncoding + 1)
	assert.Nil(t, rPool)
	assert.Error(t, err)

	wPool, err := getWriterPool(maxEncoding + 1)
	assert.Nil(t, wPool)
	assert.Error(t, err)
}
