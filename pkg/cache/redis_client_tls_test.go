package cache

import (
	"testing"

	dstls "github.com/grafana/dskit/crypto/tls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestNewRedisClient_TLSEnabled_InvalidConfig_FailsClosed pins the fail-closed
// guarantee: when an operator opts into TLS but the TLS settings cannot be
// assembled into a valid *tls.Config, NewRedisClient must return an error
// rather than silently constructing a cleartext client. A previous version
// logged the error and proceeded without TLS, which silently downgraded
// operator-intended encrypted traffic.
func TestNewRedisClient_TLSEnabled_InvalidConfig_FailsClosed(t *testing.T) {
	cfg := &RedisConfig{
		Endpoint:   "localhost:6379",
		SingleNode: true,
		TLSEnabled: true,
		TLS: dstls.ClientConfig{
			// Any value not in dskit's tlsVersions table makes GetTLSConfig
			// fail deterministically without needing filesystem fixtures.
			MinVersion: "NotARealTLSVersion",
		},
	}

	client, err := NewRedisClient(cfg, "test", prometheus.NewRegistry())
	require.Error(t, err, "TLS configuration error must surface as an error from NewRedisClient")
	require.Nil(t, client, "no client must be returned when TLS configuration fails")
	require.Contains(t, err.Error(), "TLS", "error must indicate the TLS misconfiguration to operators")
}

// TestNewRedisClient_TLSEnabled_ValidConfig_OK confirms the happy path: a
// TLSEnabled config that dskit can assemble (here, the trivial
// InsecureSkipVerify-only form) returns a client without error. Construction
// is decoupled from connection — go-redis's universal client is lazy — so
// this does not require an actual TLS endpoint.
func TestNewRedisClient_TLSEnabled_ValidConfig_OK(t *testing.T) {
	cfg := &RedisConfig{
		Endpoint:   "localhost:6379",
		SingleNode: true,
		TLSEnabled: true,
		TLS: dstls.ClientConfig{
			InsecureSkipVerify: true,
		},
	}

	client, err := NewRedisClient(cfg, "test", prometheus.NewRegistry())
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()
}

// TestNewRedisClient_TLSDisabled_IgnoresTLSBlock confirms that fields under
// the inlined TLS block are not validated when tls_enabled is false — so a
// stale or pre-populated TLS section (e.g. left over from disabling TLS)
// does not break startup.
func TestNewRedisClient_TLSDisabled_IgnoresTLSBlock(t *testing.T) {
	cfg := &RedisConfig{
		Endpoint:   "localhost:6379",
		SingleNode: true,
		TLSEnabled: false,
		TLS: dstls.ClientConfig{
			MinVersion: "NotARealTLSVersion",
		},
	}

	client, err := NewRedisClient(cfg, "test", prometheus.NewRegistry())
	require.NoError(t, err, "TLS block must be ignored when tls_enabled=false")
	require.NotNil(t, client)
	defer client.Close()
}
