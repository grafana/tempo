package generator

import (
	"context"
	"errors"

	"github.com/cortexproject/cortex/pkg/cortexpb"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

// RemoteWriteAppendable can create RemoteWriteAppender
type RemoteWriteAppendable struct {
	logger log.Logger
	userID string
	cfg    Config

	// TODO add overrides/limits

	metrics *remoteWriteMetrics
}

func newRemoteWriteAppendable(cfg Config, logger log.Logger, userID string, metrics *remoteWriteMetrics) storage.Appendable {
	if !cfg.RemoteWrite.Enabled {
		level.Info(logger).Log("msg", "remote-write is disabled")
		return &noopAppender{}
	}

	return &RemoteWriteAppendable{
		logger:  logger,
		userID:  userID,
		cfg:     cfg,
		metrics: metrics,
	}
}

type RemoteWriteAppender struct {
	logger       log.Logger
	ctx          context.Context
	remoteWriter RemoteWriter
	userID       string

	// TODO Loki uses util.EvictingQueue here to limit the amount of samples written per remote write request
	labels    []labels.Labels
	samples   []cortexpb.Sample
	examplars []cortexpb.Exemplar

	metrics *remoteWriteMetrics
}

func (a *RemoteWriteAppendable) Appender(ctx context.Context) storage.Appender {
	client, err := NewRemoteWriter(a.cfg.RemoteWrite, a.userID)
	if err != nil {
		level.Error(a.logger).Log("msg", "error creating remote-write client; setting appender as noop", "err", err, "tenant", a.userID)
		return &noopAppender{}
	}

	return &RemoteWriteAppender{
		logger:       a.logger,
		ctx:          ctx,
		remoteWriter: client,
		userID:       a.userID,
		metrics:      a.metrics,
	}
}

func (a *RemoteWriteAppender) Append(_ storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	a.labels = append(a.labels, l)
	a.samples = append(a.samples, cortexpb.Sample{
		TimestampMs: t,
		Value:       v,
	})
	return 0, nil
}

func (a *RemoteWriteAppender) AppendExemplar(storage.SeriesRef, labels.Labels, exemplar.Exemplar) (storage.SeriesRef, error) {
	// TODO as a tracing backend, we should definitely support this 😅
	return 0, errors.New("exemplars are unsupported")
}

func (a *RemoteWriteAppender) Commit() error {
	level.Debug(a.logger).Log("msg", "writing samples to remote_write target", "tenant", a.userID, "target", a.remoteWriter.Endpoint(), "count", len(a.samples))
	a.metrics.samplesSent.WithLabelValues(a.userID).Set(float64(len(a.samples)))
	a.metrics.remoteWriteTotal.WithLabelValues(a.userID).Inc()

	// TODO is cortexpb.RULE appropriate here? alternative is API...
	req := cortexpb.ToWriteRequest(a.labels, a.samples, nil, cortexpb.RULE)
	defer cortexpb.ReuseSlice(req.Timeseries)

	reqBytes, err := req.Marshal()
	if err != nil {
		return err
	}
	reqBytes = snappy.Encode(nil, reqBytes)

	// TODO I have seen requests fail because the message was to big (there is a limit of 10MB I think?), should we limit or split messages here?

	err = a.remoteWriter.Store(a.ctx, reqBytes)
	// TODO the returned error can be of type RecoverableError with a retryAfter duration, should we do something with this?
	if err != nil {
		level.Error(a.logger).Log("msg", "could not store metrics-generator samples", "tenant", a.userID, "err", err)
		a.metrics.remoteWriteErrors.WithLabelValues(a.userID).Inc()
		return err
	}

	a.labels = nil
	a.samples = nil
	return nil
}

func (a *RemoteWriteAppender) Rollback() error {
	a.labels = nil
	a.samples = nil
	return nil
}

// noopAppender implements storage.Appendable and storage.Appender
type noopAppender struct{}

var _ storage.Appendable = (*noopAppender)(nil)
var _ storage.Appender = (*noopAppender)(nil)

func (a *noopAppender) Appender(_ context.Context) storage.Appender {
	return a
}

func (a *noopAppender) Append(_ storage.SeriesRef, _ labels.Labels, _ int64, _ float64) (storage.SeriesRef, error) {
	return 0, nil
}

func (a *noopAppender) AppendExemplar(_ storage.SeriesRef, _ labels.Labels, _ exemplar.Exemplar) (storage.SeriesRef, error) {
	return 0, nil
}

func (a *noopAppender) Commit() error {
	return nil
}

func (a *noopAppender) Rollback() error {
	return nil
}
