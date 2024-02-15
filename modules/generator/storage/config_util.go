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
func generateTenantRemoteWriteConfigs(originalCfgs []prometheus_config.RemoteWriteConfig, tenant string, headers map[string]string, addOrgIDHeader bool, logger log.Logger) []*prometheus_config.RemoteWriteConfig {
	var cloneCfgs []*prometheus_config.RemoteWriteConfig

	for _, originalCfg := range originalCfgs {
		cloneCfg := &prometheus_config.RemoteWriteConfig{}
		*cloneCfg = originalCfg

		// Inject/overwrite X-Scope-OrgID header in multi-tenant setups
		if tenant != util.FakeTenantID && addOrgIDHeader {
			// Copy headers so we can modify them
			cloneCfg.Headers = copyMap(cloneCfg.Headers)

			// Ensure that no variation of the X-Scope-OrgId header can be added, which might trick authentication
			for k, v := range cloneCfg.Headers {
				if strings.EqualFold(user.OrgIDHeaderName, strings.TrimSpace(k)) {
					level.Warn(logger).Log("msg", "discarding X-Scope-OrgId header", "key", k, "value", v)
					delete(cloneCfg.Headers, k)
				}
			}

			cloneCfg.Headers[user.OrgIDHeaderName] = tenant
		}

		// Inject/overwrite custom headers
		// Caution! This can overwrite the X-Scope-OrgID header
		for k, v := range headers {
			cloneCfg.Headers[k] = v
		}

		cloneCfgs = append(cloneCfgs, cloneCfg)
	}

	return cloneCfgs
}

// copyMap creates a new map containing all values from the given map.
func copyMap(m map[string]string) map[string]string {
	newMap := make(map[string]string, len(m))

	for k, v := range m {
		newMap[k] = v
	}

	return newMap
}
