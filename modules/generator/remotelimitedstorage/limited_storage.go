package remotelimitedstorage

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerclient"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/statsd_exporter/pkg/level"
)

type LimitedStorage struct {
	storage      storage.Appendable
	usageTracker *usagetrackerclient.UsageTrackerClient
	tenant       string
	logger       log.Logger
}

var _ storage.Appendable = (*LimitedStorage)(nil)

func NewLimitedStorage(storage storage.Appendable, usageTracker *usagetrackerclient.UsageTrackerClient, tenant string, logger log.Logger) *LimitedStorage {
	return &LimitedStorage{storage: storage, usageTracker: usageTracker, tenant: tenant, logger: log.With(logger, "tenant", tenant)}
}

func (l *LimitedStorage) Appender(ctx context.Context) storage.Appender {
	return &limitedAppender{
		logger:          l.logger,
		appender:        l.storage.Appender(ctx),
		usageTracker:    l.usageTracker,
		tenant:          l.tenant,
		capturedAppends: make(map[uint64][]capturedAppend),
	}
}

type limitedAppender struct {
	logger       log.Logger
	appender     storage.Appender
	usageTracker *usagetrackerclient.UsageTrackerClient
	tenant       string

	capturedAppends map[uint64][]capturedAppend
}

type appendType int

const (
	appendTypeSample appendType = iota
	appendTypeCTZeroSample
	appendTypeExemplar
	appendTypeHistogram
	appendTypeHistogramCTZeroSample
	appendTypeMetadata
)

type capturedAppend struct {
	ref  storage.SeriesRef
	lbls labels.Labels
	typ  appendType
	t    int64
	v    float64
	ct   int64
	h    *histogram.Histogram
	fh   *histogram.FloatHistogram
	e    exemplar.Exemplar
	m    metadata.Metadata
}

func hashLabels(lbls labels.Labels) uint64 {
	var buf [1024]byte
	hash, _ := lbls.HashWithoutLabels(buf[:0], "__metrics_gen_instance")
	return hash
}

// Append implements storage.Appender.
func (l *limitedAppender) Append(ref storage.SeriesRef, lbls labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	ca := capturedAppend{
		ref:  ref,
		lbls: lbls,
		typ:  appendTypeSample,
		t:    t,
		v:    v,
	}
	hash := hashLabels(lbls)
	l.capturedAppends[hash] = append(l.capturedAppends[hash], ca)
	return ref, nil
}

// AppendCTZeroSample implements storage.Appender.
func (l *limitedAppender) AppendCTZeroSample(ref storage.SeriesRef, lbls labels.Labels, t int64, ct int64) (storage.SeriesRef, error) {
	ca := capturedAppend{
		ref:  ref,
		lbls: lbls,
		typ:  appendTypeCTZeroSample,
		t:    t,
		ct:   ct,
	}
	hash := hashLabels(lbls)
	l.capturedAppends[hash] = append(l.capturedAppends[hash], ca)
	return ref, nil
}

// AppendExemplar implements storage.Appender.
func (l *limitedAppender) AppendExemplar(ref storage.SeriesRef, lbls labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	ca := capturedAppend{
		ref:  ref,
		lbls: lbls,
		typ:  appendTypeExemplar,
		e:    e,
	}
	hash := hashLabels(lbls)
	l.capturedAppends[hash] = append(l.capturedAppends[hash], ca)
	return ref, nil
}

// AppendHistogram implements storage.Appender.
func (l *limitedAppender) AppendHistogram(ref storage.SeriesRef, lbls labels.Labels, t int64, h *histogram.Histogram, fh *histogram.FloatHistogram) (storage.SeriesRef, error) {
	ca := capturedAppend{
		ref:  ref,
		lbls: lbls,
		typ:  appendTypeHistogram,
		t:    t,
		h:    h,
		fh:   fh,
	}
	hash := hashLabels(lbls)
	l.capturedAppends[hash] = append(l.capturedAppends[hash], ca)
	return ref, nil
}

// AppendHistogramCTZeroSample implements storage.Appender.
func (l *limitedAppender) AppendHistogramCTZeroSample(ref storage.SeriesRef, lbls labels.Labels, t int64, ct int64, h *histogram.Histogram, fh *histogram.FloatHistogram) (storage.SeriesRef, error) {
	ca := capturedAppend{
		ref:  ref,
		lbls: lbls,
		typ:  appendTypeHistogramCTZeroSample,
		t:    t,
		ct:   ct,
		h:    h,
		fh:   fh,
	}
	hash := hashLabels(lbls)
	l.capturedAppends[hash] = append(l.capturedAppends[hash], ca)
	return ref, nil
}

// Commit implements storage.Appender.
func (l *limitedAppender) Commit() error {

	numCaptured := 0
	numHashes := len(l.capturedAppends)
	for _, cas := range l.capturedAppends {
		numCaptured += len(cas)
	}

	hashes := make([]uint64, 0, len(l.capturedAppends))
	for hash := range l.capturedAppends {
		hashes = append(hashes, hash)
	}
	ctx := user.InjectOrgID(context.Background(), l.tenant)
	rejected, err := l.usageTracker.TrackSeries(ctx, l.tenant, hashes)
	if err != nil {
		return err
	}

	for _, hash := range rejected {
		delete(l.capturedAppends, hash)
	}

	level.Warn(l.logger).Log("msg", "remote limiting applied", "rejected_series", len(rejected), "series_before", numHashes, "series_after", len(l.capturedAppends))

	for _, cas := range l.capturedAppends {
		for _, ca := range cas {
			switch ca.typ {
			case appendTypeSample:
				l.appender.Append(ca.ref, ca.lbls, ca.t, ca.v)
			case appendTypeCTZeroSample:
				l.appender.AppendCTZeroSample(ca.ref, ca.lbls, ca.t, ca.ct)
			case appendTypeExemplar:
				l.appender.AppendExemplar(ca.ref, ca.lbls, ca.e)
			case appendTypeHistogram:
				l.appender.AppendHistogram(ca.ref, ca.lbls, ca.t, ca.h, ca.fh)
			case appendTypeHistogramCTZeroSample:
				l.appender.AppendHistogramCTZeroSample(ca.ref, ca.lbls, ca.t, ca.ct, ca.h, ca.fh)
			case appendTypeMetadata:
				l.appender.UpdateMetadata(ca.ref, ca.lbls, ca.m)
			default:
				panic(fmt.Sprintf("unknown append type: %d", ca.typ))
			}
		}
	}

	return l.appender.Commit()
}

// Rollback implements storage.Appender.
func (l *limitedAppender) Rollback() error {
	return l.appender.Rollback()
}

// SetOptions implements storage.Appender.
func (l *limitedAppender) SetOptions(opts *storage.AppendOptions) {
	l.appender.SetOptions(opts)
}

// UpdateMetadata implements storage.Appender.
func (l *limitedAppender) UpdateMetadata(ref storage.SeriesRef, lbls labels.Labels, m metadata.Metadata) (storage.SeriesRef, error) {
	ca := capturedAppend{
		ref:  ref,
		lbls: lbls,
		typ:  appendTypeMetadata,
		m:    m,
	}
	hash := hashLabels(lbls)
	l.capturedAppends[hash] = append(l.capturedAppends[hash], ca)
	return ref, nil
}

var _ storage.Appender = (*limitedAppender)(nil)
