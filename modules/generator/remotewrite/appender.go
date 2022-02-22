package remotewrite

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/klauspost/compress/snappy"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage"
)

// TODO: make this configurable
var maxWriteRequestSize = 3 * 1024 * 1024 // 3MB

// remoteWriteAppender is a storage.Appender that remote writes samples and exemplars.
type remoteWriteAppender struct {
	logger       log.Logger
	ctx          context.Context
	remoteWriter *remoteWriteClient
	userID       string

	// TODO Loki uses util.EvictingQueue here to limit the amount of samples written per remote write request
	labels         [][]prompb.Label
	samples        []prompb.Sample
	exemplarLabels [][]prompb.Label
	exemplars      []prompb.Exemplar

	metrics *Metrics
}

var _ storage.Appender = (*remoteWriteAppender)(nil)

func (a *remoteWriteAppender) Append(_ storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	a.labels = append(a.labels, labelsToLabelsProto(l))
	a.samples = append(a.samples, prompb.Sample{
		Timestamp: t,
		Value:     v,
	})
	return 0, nil
}

func (a *remoteWriteAppender) AppendExemplar(_ storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	a.exemplarLabels = append(a.exemplarLabels, labelsToLabelsProto(l))
	a.exemplars = append(a.exemplars, prompb.Exemplar{
		Labels:    labelsToLabelsProto(e.Labels),
		Value:     e.Value,
		Timestamp: e.Ts,
	})
	return 0, nil
}

func (a *remoteWriteAppender) Commit() error {
	level.Debug(a.logger).Log("msg", "writing samples to remote_write target", "tenant", a.userID, "target", a.remoteWriter.Endpoint(), "count", len(a.samples))

	if len(a.samples) == 0 {
		return nil
	}

	reqs := a.buildRequests()

	a.metrics.samplesSent.WithLabelValues(a.userID).Add(float64(len(a.samples)))
	a.metrics.exemplarsSent.WithLabelValues(a.userID).Add(float64(len(a.exemplars)))
	a.metrics.remoteWriteTotal.WithLabelValues(a.userID).Add(float64(len(reqs)))

	err := a.sendRequests(reqs)
	if err != nil {
		level.Error(a.logger).Log("msg", "error sending remote-write requests", "tenant", a.userID, "target", a.remoteWriter.Endpoint(), "err", err)
		a.metrics.remoteWriteErrors.WithLabelValues(a.userID).Inc()
		return err
	}

	a.clearBuffers()
	return nil
}

// buildRequests builds a slice of prompb.WriteRequest of which each requests has a maximum size of
// maxWriteRequestSize (uncompressed).
func (a *remoteWriteAppender) buildRequests() []*prompb.WriteRequest {
	var requests []*prompb.WriteRequest
	currentRequest := &prompb.WriteRequest{}

	appendTimeSeries := func(ts prompb.TimeSeries) {
		if currentRequest.Size()+ts.Size() >= maxWriteRequestSize {
			requests = append(requests, currentRequest)
			currentRequest = &prompb.WriteRequest{}
		}
		currentRequest.Timeseries = append(currentRequest.Timeseries, ts)
	}

	for i, s := range a.samples {
		appendTimeSeries(prompb.TimeSeries{
			Labels:  a.labels[i],
			Samples: []prompb.Sample{s},
		})
	}

	for i, e := range a.exemplars {
		appendTimeSeries(prompb.TimeSeries{
			Labels:    a.exemplarLabels[i],
			Exemplars: []prompb.Exemplar{e},
		})
	}

	if len(currentRequest.Timeseries) != 0 {
		requests = append(requests, currentRequest)
	}

	return requests
}

func (a *remoteWriteAppender) sendRequests(reqs []*prompb.WriteRequest) error {
	for _, req := range reqs {
		err := a.sendWriteRequest(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *remoteWriteAppender) sendWriteRequest(req *prompb.WriteRequest) error {
	bytes, err := req.Marshal()
	if err != nil {
		return err
	}
	bytes = snappy.Encode(nil, bytes)

	// TODO the returned error can be of type RecoverableError with a retryAfter duration, should we do something with this?
	return a.remoteWriter.Store(a.ctx, bytes)
}

func (a *remoteWriteAppender) Rollback() error {
	a.clearBuffers()
	return nil
}

func (a *remoteWriteAppender) clearBuffers() {
	a.labels = nil
	a.samples = nil
	a.exemplars = nil
	a.exemplarLabels = nil
}

func labelsToLabelsProto(labels labels.Labels) []prompb.Label {
	result := make([]prompb.Label, len(labels))
	for i, l := range labels {
		result[i] = prompb.Label{
			Name:  l.Name,
			Value: l.Value,
		}
	}
	return result
}
