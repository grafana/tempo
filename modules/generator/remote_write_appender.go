package generator

import (
	"context"
	"errors"

	"github.com/cortexproject/cortex/pkg/cortexpb"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

// TODO: Configure this in the config file.
var maxWriteRequestSize = 3 * 1024 * 1024 // 3MB

// remoteWriteAppendable is a Prometheus storage.Appendable that remote writes samples.
type remoteWriteAppendable struct {
	logger   log.Logger
	tenantID string
	cfg      *Config

	// TODO add overrides/limits

	metrics *remoteWriteMetrics
}

var _ storage.Appendable = (*remoteWriteAppendable)(nil)

// newRemoteWriteAppendable creates a Prometheus storage.Appendable that can remote write. If tenantID is not empty, it sets
// the X-Scope-Orgid header on every request.
func newRemoteWriteAppendable(cfg *Config, logger log.Logger, tenantID string, metrics *remoteWriteMetrics) storage.Appendable {
	if !cfg.RemoteWrite.Enabled {
		level.Info(logger).Log("msg", "remote-write is disabled")
		return &noopAppender{}
	}

	return &remoteWriteAppendable{
		logger:   logger,
		tenantID: tenantID,
		cfg:      cfg,
		metrics:  metrics,
	}
}

func (a *remoteWriteAppendable) Appender(ctx context.Context) storage.Appender {
	client, err := newRemoteWriteClient(&a.cfg.RemoteWrite.Client, a.tenantID)
	if err != nil {
		level.Error(a.logger).Log("msg", "error creating remote-write client; setting appender as noop", "err", err, "tenant", a.tenantID)
		return &noopAppender{}
	}

	return &remoteWriteAppender{
		logger:       a.logger,
		ctx:          ctx,
		remoteWriter: client,
		userID:       a.tenantID,
		metrics:      a.metrics,
	}
}

// remoteWriteAppender is a storage.Appender that remote writes samples.
type remoteWriteAppender struct {
	logger       log.Logger
	ctx          context.Context
	remoteWriter *remoteWriteClient
	userID       string

	// TODO Loki uses util.EvictingQueue here to limit the amount of samples written per remote write request
	labels    []labels.Labels
	samples   []cortexpb.Sample
	examplars []cortexpb.Exemplar

	metrics *remoteWriteMetrics
}

var _ storage.Appender = (*remoteWriteAppender)(nil)

func (a *remoteWriteAppender) Append(_ storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	a.labels = append(a.labels, l)
	a.samples = append(a.samples, cortexpb.Sample{
		TimestampMs: t,
		Value:       v,
	})
	return 0, nil
}

func (a *remoteWriteAppender) AppendExemplar(storage.SeriesRef, labels.Labels, exemplar.Exemplar) (storage.SeriesRef, error) {
	// TODO as a tracing backend, we should definitely support this 😅
	return 0, errors.New("exemplars are unsupported")
}

func (a *remoteWriteAppender) Commit() error {
	level.Debug(a.logger).Log("msg", "writing samples to remote_write target", "tenant", a.userID, "target", a.remoteWriter.Endpoint(), "count", len(a.samples))

	if len(a.samples) == 0 {
		return nil
	}

	reqs := a.buildRequests()

	a.metrics.samplesSent.WithLabelValues(a.userID).Add(float64(len(a.samples)))
	a.metrics.remoteWriteTotal.WithLabelValues(a.userID).Add(float64(len(reqs)))

	err := a.sendWriteRequests(reqs)
	if err != nil {
		level.Error(a.logger).Log("msg", "error sending remote-write requests", "tenant", a.userID, "target", a.remoteWriter.Endpoint(), "err", err)
		a.metrics.remoteWriteErrors.WithLabelValues(a.userID).Inc()
		return err
	}

	a.labels = nil
	a.samples = nil
	return nil
}

// buildRequests builds a slice of *cortexpb.WriteRequest
// It builds requests with a maximum size of maxWriteRequestSize (uncompressed).
func (a *remoteWriteAppender) buildRequests() []*cortexpb.WriteRequest {
	var requests []*cortexpb.WriteRequest
	currentRequest := newWriteRequest()

	for i, s := range a.samples {
		ts := cortexpb.TimeseriesFromPool()
		ts.Samples = append(ts.Samples, s)
		ts.Labels = append(ts.Labels, cortexpb.FromLabelsToLabelAdapters(a.labels[i])...)

		if currentRequest.Size()+ts.Size() >= maxWriteRequestSize {
			requests = append(requests, currentRequest)
			currentRequest = newWriteRequest()
		}

		currentRequest.Timeseries = append(currentRequest.Timeseries, cortexpb.PreallocTimeseries{TimeSeries: ts})
	}

	if len(currentRequest.Timeseries) != 0 {
		requests = append(requests, currentRequest)
	}

	return requests
}

func (a *remoteWriteAppender) sendWriteRequests(reqs []*cortexpb.WriteRequest) error {
	for _, req := range reqs {
		err := a.sendWriteRequest(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *remoteWriteAppender) sendWriteRequest(req *cortexpb.WriteRequest) error {
	defer cortexpb.ReuseSlice(req.Timeseries)
	reqBytes, err := req.Marshal()
	if err != nil {
		return err
	}
	reqBytes = snappy.Encode(nil, reqBytes)

	// TODO the returned error can be of type RecoverableError with a retryAfter duration, should we do something with this?
	return a.remoteWriter.Store(a.ctx, reqBytes)
}

func newWriteRequest() *cortexpb.WriteRequest {
	return &cortexpb.WriteRequest{
		Timeseries: cortexpb.PreallocTimeseriesSliceFromPool(),
		Source:     cortexpb.API,
	}
}

func (a *remoteWriteAppender) Rollback() error {
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

type remoteWriteMetrics struct {
	samplesSent       *prometheus.CounterVec
	remoteWriteErrors *prometheus.CounterVec
	remoteWriteTotal  *prometheus.CounterVec
}

func newRemoteWriteMetrics(reg prometheus.Registerer) *remoteWriteMetrics {
	return &remoteWriteMetrics{
		samplesSent: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_samples_sent_total",
			Help:      "Number of samples sent",
		}, []string{"tenant"}),
		remoteWriteErrors: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_remote_write_errors",
			Help:      "Number of remote-write requests that failed due to error.",
		}, []string{"tenant"}),
		remoteWriteTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_remote_write_total",
			Help:      "Number of remote-write requests.",
		}, []string{"tenant"}),
	}
}
