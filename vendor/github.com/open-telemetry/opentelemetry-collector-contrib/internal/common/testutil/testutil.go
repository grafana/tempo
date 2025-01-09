// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/featuregate"
)

type portpair struct {
	first string
	last  string
}

// GetAvailableLocalAddress finds an available local port on tcp network and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalAddress(tb testing.TB) string {
	return GetAvailableLocalNetworkAddress(tb, "tcp")
}

// GetAvailableLocalNetworkAddress finds an available local port on specified network and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalNetworkAddress(tb testing.TB, network string) string {
	// Retry has been added for windows as net.Listen can return a port that is not actually available. Details can be
	// found in https://github.com/docker/for-win/issues/3171 but to summarize Hyper-V will reserve ranges of ports
	// which do not show up under the "netstat -ano" but can only be found by
	// "netsh interface ipv4 show excludedportrange protocol=tcp".  We'll use []exclusions to hold those ranges and
	// retry if the port returned by GetAvailableLocalAddress falls in one of those them.
	var exclusions []portpair

	portFound := false
	if runtime.GOOS == "windows" {
		exclusions = getExclusionsList(tb)
	}

	var endpoint string
	for !portFound {
		endpoint = findAvailableAddress(tb, network)
		_, port, err := net.SplitHostPort(endpoint)
		require.NoError(tb, err)
		portFound = true
		if runtime.GOOS == "windows" {
			for _, pair := range exclusions {
				if port >= pair.first && port <= pair.last {
					portFound = false
					break
				}
			}
		}
	}

	return endpoint
}

func findAvailableAddress(tb testing.TB, network string) string {
	switch network {
	// net.Listen supported network strings
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		ln, err := net.Listen(network, "localhost:0")
		require.NoError(tb, err, "Failed to get a free local port")
		// There is a possible race if something else takes this same port before
		// the test uses it, however, that is unlikely in practice.
		defer func() {
			assert.NoError(tb, ln.Close())
		}()
		return ln.Addr().String()
	// net.ListenPacket supported network strings
	case "udp", "udp4", "udp6", "unixgram":
		ln, err := net.ListenPacket(network, "localhost:0")
		require.NoError(tb, err, "Failed to get a free local port")
		// There is a possible race if something else takes this same port before
		// the test uses it, however, that is unlikely in practice.
		defer func() {
			assert.NoError(tb, ln.Close())
		}()
		return ln.LocalAddr().String()
	}
	return ""
}

// Get excluded ports on Windows from the command: netsh interface ipv4 show excludedportrange protocol=tcp
func getExclusionsList(tb testing.TB) []portpair {
	cmdTCP := exec.Command("netsh", "interface", "ipv4", "show", "excludedportrange", "protocol=tcp")
	outputTCP, errTCP := cmdTCP.CombinedOutput()
	require.NoError(tb, errTCP)
	exclusions := createExclusionsList(tb, string(outputTCP))

	cmdUDP := exec.Command("netsh", "interface", "ipv4", "show", "excludedportrange", "protocol=udp")
	outputUDP, errUDP := cmdUDP.CombinedOutput()
	require.NoError(tb, errUDP)
	exclusions = append(exclusions, createExclusionsList(tb, string(outputUDP))...)

	return exclusions
}

func createExclusionsList(tb testing.TB, exclusionsText string) []portpair {
	var exclusions []portpair

	parts := strings.Split(exclusionsText, "--------")
	require.Len(tb, parts, 3)
	portsText := strings.Split(parts[2], "*")
	require.Greater(tb, len(portsText), 1) // original text may have a suffix like " - Administered port exclusions."
	lines := strings.Split(portsText[0], "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			entries := strings.Fields(strings.TrimSpace(line))
			require.Len(tb, entries, 2)
			pair := portpair{entries[0], entries[1]}
			exclusions = append(exclusions, pair)
		}
	}
	return exclusions
}

// Force the state of feature gate for a test
// usage: defer SetFeatureGateForTest("gateName", true)()
func SetFeatureGateForTest(tb testing.TB, gate *featuregate.Gate, enabled bool) func() {
	originalValue := gate.IsEnabled()
	require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	return func() {
		require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), originalValue))
	}
}

func GetAvailablePort(tb testing.TB) int {
	endpoint := GetAvailableLocalAddress(tb)
	_, port, err := net.SplitHostPort(endpoint)
	require.NoError(tb, err)

	portInt, err := strconv.Atoi(port)
	require.NoError(tb, err)

	return portInt
}

// EndpointForPort gets the endpoint for a given port using localhost.
func EndpointForPort(port int) string {
	return fmt.Sprintf("localhost:%d", port)
}
