package frontend

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/tempodb/backend"
)

// extractTenant extracts tenant ID from request context and returns HTTP error response if extraction fails
func extractTenant(req *http.Request, logger log.Logger) (string, *http.Response) {
	tenant, err := user.ExtractOrgID(req.Context())
	if err != nil {
		level.Error(logger).Log("msg", "failed to extract tenant id", "err", err)
		return "", &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}
	}
	return tenant, nil
}

func rf1FilterFn(rf1After time.Time) func(m *backend.BlockMeta) bool {
	return func(m *backend.BlockMeta) bool {
		if rf1After.IsZero() {
			return m.ReplicationFactor == backend.DefaultReplicationFactor
		}

		return (m.ReplicationFactor == backend.DefaultReplicationFactor && m.StartTime.Before(rf1After)) ||
			(m.ReplicationFactor == backend.MetricsGeneratorReplicationFactor && m.StartTime.After(rf1After))
	}
}
