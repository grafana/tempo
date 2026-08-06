package main

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchedulerTLSConfig(t *testing.T) {
	for _, tc := range []struct {
		name       string
		serverName string
		ca         string
		minVersion string
		wantMin    uint16
		wantErr    string
	}{
		{
			name:       "default is the strongest version",
			minVersion: defaultTLSMinVersion,
			wantMin:    tls.VersionTLS13,
		},
		{
			name:       "an older version can be requested explicitly",
			minVersion: "VersionTLS12",
			wantMin:    tls.VersionTLS12,
		},
		{
			name:       "server name is carried through for SNI",
			serverName: "scheduler.example.com",
			minVersion: defaultTLSMinVersion,
			wantMin:    tls.VersionTLS13,
		},
		{
			name:       "an unknown version name is rejected, not silently defaulted",
			minVersion: "TLS1.3",
			wantErr:    "unknown minimum TLS version",
		},
		{
			name:       "the error names the versions that are allowed",
			minVersion: "",
			wantErr:    defaultTLSMinVersion,
		},
		{
			name:       "a missing CA file is reported",
			ca:         "/nonexistent/ca.pem",
			minVersion: defaultTLSMinVersion,
			wantErr:    "reading CA cert",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := schedulerTLSConfig(tc.serverName, tc.ca, tc.minVersion)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantMin, cfg.MinVersion, "the requested minimum version must reach the TLS config")
			require.Equal(t, tc.serverName, cfg.ServerName)
			require.NotNil(t, cfg.RootCAs)
		})
	}
}

// TestSchedulerTransportCredentialsWithoutTLS covers the no-TLS path. TLS settings supplied without
// --tls used to be discarded silently, so `--tls-ca ca.pem` (which reads like "use TLS with this CA")
// sent the tenant header and a destructive control-plane call over cleartext with no signal. They are
// now refused instead of downgraded, and the version name is validated either way so a typo cannot
// hide behind a plaintext connection.
func TestSchedulerTransportCredentialsWithoutTLS(t *testing.T) {
	t.Run("plain connection when no TLS settings are given", func(t *testing.T) {
		creds, err := schedulerTransportCredentials(false, "", "", defaultTLSMinVersion)
		require.NoError(t, err)
		require.Equal(t, "insecure", creds.Info().SecurityProtocol)
	})

	t.Run("a CA without --tls is refused rather than ignored", func(t *testing.T) {
		_, err := schedulerTransportCredentials(false, "", "/tmp/ca.pem", defaultTLSMinVersion)
		require.ErrorContains(t, err, "--tls")
	})

	t.Run("a server name without --tls is refused rather than ignored", func(t *testing.T) {
		_, err := schedulerTransportCredentials(false, "scheduler.example.com", "", defaultTLSMinVersion)
		require.ErrorContains(t, err, "--tls")
	})

	t.Run("an invalid version is rejected even without --tls", func(t *testing.T) {
		_, err := schedulerTransportCredentials(false, "", "", "TLS1.3")
		require.ErrorContains(t, err, "unknown minimum TLS version")
	})
}
