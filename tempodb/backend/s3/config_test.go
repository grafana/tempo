package s3

import (
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSSEConfigEncryptionKeyRedacted(t *testing.T) {
	const plaintext = "my-super-secret-key"

	var key flagext.Secret
	require.NoError(t, key.Set(plaintext))

	cfg := SSEConfig{
		Type:                  SSEC,
		CustomerEncryptionKey: key,
	}

	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	assert.NotContains(t, string(out), plaintext, "plaintext encryption key must not appear in YAML output")
	assert.Contains(t, string(out), "********", "redacted marker must appear in YAML output")
}
