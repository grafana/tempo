package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromVersionErrors(t *testing.T) {
	encoding, err := FromVersion("definitely-not-a-real-version")
	assert.Error(t, err)
	assert.Nil(t, encoding)
}

func TestAllVersions(t *testing.T) {
	for _, v := range AllEncodings() {
		encoding, err := FromVersion(v.Version())

		require.Equal(t, v.Version(), encoding.Version())
		require.NoError(t, err)
	}
}
