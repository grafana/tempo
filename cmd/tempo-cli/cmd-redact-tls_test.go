package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchedulerTransportCredentialsTLSMinVersion(t *testing.T) {
	// Valid version names are accepted (default is the strongest).
	_, err := schedulerTransportCredentials(true, "", "", "VersionTLS13")
	require.NoError(t, err)
	_, err = schedulerTransportCredentials(true, "", "", "VersionTLS12")
	require.NoError(t, err)

	// An unknown version name is rejected rather than silently ignored.
	_, err = schedulerTransportCredentials(true, "", "", "TLS1.3")
	require.Error(t, err)

	// The insecure (no-TLS) path ignores the min version entirely.
	_, err = schedulerTransportCredentials(false, "", "", "")
	require.NoError(t, err)
}
