package storage

import (
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	prometheus_config "github.com/prometheus/prometheus/config"

	"github.com/grafana/tempo/pkg/util"
)

// generateTenantRemoteWriteConfigs creates a copy of the remote write configurations with the
// X-Scope-OrgID header present for the given tenant, unless Tempo is run in single tenant mode or instructed not to add X-Scope-OrgID header.
func generateTenantRemoteWriteConfigs(inputs []prometheus_config.RemoteWriteConfig, tenant string, headers map[string]string, addOrgIDHeader bool, logger log.Logger, sendNativeHistograms bool) []*prometheus_config.RemoteWriteConfig {
	var outputs []*prometheus_config.RemoteWriteConfig

	for _, input := range inputs {
		output := &prometheus_config.RemoteWriteConfig{}
		*output = input

		// Copy headers so we can modify them
		output.Headers = copyMap(output.Headers)

		// Inject/overwrite custom headers from runtime overrides
		for k, v := range headers {
			output.Headers[k] = v
		}

		// Inject/overwrite X-Scope-OrgID header in multi-tenant setups
		if tenant != util.FakeTenantID && addOrgIDHeader {
			existing := ""
			for k, v := range output.Headers {
				if strings.EqualFold(user.OrgIDHeaderName, k) {
					existing = v
					break
				}
			}

			if existing == "" {
				output.Headers[user.OrgIDHeaderName] = tenant
			} else {
				// Remote write config already contains the header, so we don't overwrite it.
				level.Warn(logger).Log("msg", "underlying remote write already contains X-Scope-OrgId header, not applying new value",
					"remoteWriteName", input.Name,
					"remoteWriteURL", input.URL,
					"existing", existing,
					"new", tenant)
			}
		}

		output.SendNativeHistograms = sendNativeHistograms
		// TODO: enable exemplars
		// cloneCfg.SendExemplars = sendExemplars

		outputs = append(outputs, output)
	}

	return outputs
}

// copyMap creates a new map containing all values from the given map.
func copyMap(m map[string]string) map[string]string {
	newMap := make(map[string]string, len(m))

	for k, v := range m {
		newMap[k] = v
	}

	return newMap
}
