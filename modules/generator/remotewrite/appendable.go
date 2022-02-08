package remotewrite

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/prometheus/storage"
)

// remoteWriteAppendable is a Prometheus storage.Appendable that remote writes samples and exemplars.
type remoteWriteAppendable struct {
	logger   log.Logger
	tenantID string
	cfg      *Config

	// TODO add overrides/limits

	metrics *Metrics
}

var _ storage.Appendable = (*remoteWriteAppendable)(nil)

// NewAppendable creates a Prometheus storage.Appendable that can remote write. If
// tenantID is not empty, it sets the X-Scope-Orgid header on every request.
func NewAppendable(cfg *Config, logger log.Logger, tenantID string, metrics *Metrics) storage.Appendable {
	if !cfg.Enabled {
		level.Info(logger).Log("msg", "remote-write is disabled")
		return &NoopAppender{}
	}

	return &remoteWriteAppendable{
		logger:   logger,
		tenantID: tenantID,
		cfg:      cfg,
		metrics:  metrics,
	}
}

func (a *remoteWriteAppendable) Appender(ctx context.Context) storage.Appender {
	client, err := newRemoteWriteClient(&a.cfg.Client, a.tenantID)
	if err != nil {
		level.Error(a.logger).Log("msg", "error creating remote-write client; setting appender as noop", "err", err, "tenant", a.tenantID)
		return &NoopAppender{}
	}

	return &remoteWriteAppender{
		logger:       a.logger,
		ctx:          ctx,
		remoteWriter: client,
		userID:       a.tenantID,
		metrics:      a.metrics,
	}
}
