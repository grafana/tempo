package v1

import (
	"fmt"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

func TestGetPool(t *testing.T) {
	for _, enc := range backend.SupportedEncoding {
		t.Run(fmt.Sprintf("testing %s", enc), func(t *testing.T) {
			rPool, err := getReaderPool(enc)
			assert.NotNil(t, rPool)
			assert.NoError(t, err)
			assert.Equal(t, enc, rPool.Encoding())

			wPool, err := getWriterPool(enc)
			assert.NotNil(t, wPool)
			assert.NoError(t, err)
			assert.Equal(t, enc, wPool.Encoding())
		})
	}

	rPool, err := getReaderPool(maxEncoding + 1)
	assert.Nil(t, rPool)
	assert.Error(t, err)

	wPool, err := getWriterPool(maxEncoding + 1)
	assert.Nil(t, wPool)
	assert.Error(t, err)
}
