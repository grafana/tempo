package frontend

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

func rf1FilterFn(rf1After time.Time) func(m *backend.BlockMeta) bool {
	return func(m *backend.BlockMeta) bool {
		if rf1After.IsZero() {
			return m.ReplicationFactor == backend.DefaultReplicationFactor
		}

		return (m.ReplicationFactor == backend.DefaultReplicationFactor && m.StartTime.Before(rf1After)) ||
			(m.ReplicationFactor == backend.MetricsGeneratorReplicationFactor && m.StartTime.After(rf1After))
	}
}
