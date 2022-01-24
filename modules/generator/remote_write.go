package generator

import (
	"fmt"

	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/storage/remote"

	"github.com/grafana/tempo/cmd/tempo/build"
)

const (
	userAgentHeader   = "User-Agent"
	xScopeOrgIDHeader = "X-Scope-Orgid"
)

var remoteWriteUserAgent = fmt.Sprintf("tempo-remote-write/%s", build.Version)

type remoteWriteClient struct {
	remote.WriteClient
}

// newRemoteWriteClient creates a Prometheus remote.WriteClient. If tenantID is not empty, it sets
// the X-Scope-Orgid header on every request.
func newRemoteWriteClient(cfg *config.RemoteWriteConfig, tenantID string) (*remoteWriteClient, error) {
	headers := copyMap(cfg.Headers)
	headers[userAgentHeader] = remoteWriteUserAgent
	if tenantID != "" {
		headers[xScopeOrgIDHeader] = tenantID
	}

	writeClient, err := remote.NewWriteClient(
		"metrics_generator",
		&remote.ClientConfig{
			URL:              cfg.URL,
			Timeout:          cfg.RemoteTimeout,
			HTTPClientConfig: cfg.HTTPClientConfig,
			Headers:          headers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not create remote-write client for tenant: %s", tenantID)
	}

	return &remoteWriteClient{
		WriteClient: writeClient,
	}, nil
}

// copyMap creates a new map containing all values from the given map.
func copyMap(m map[string]string) map[string]string {
	newMap := make(map[string]string, len(m))

	for k, v := range m {
		newMap[k] = v
	}

	return newMap
}
