package s3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigDefaultChecksumType(t *testing.T) {
	cfg := Config{}
	checksumType := cfg.checksumType()
	// we assume by default no checksum is set
	require.False(t, checksumType.IsSet(), "default behavior changed")
}
