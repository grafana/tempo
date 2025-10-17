package remotelimitedstorage

import (
	"context"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/generator/remoteserieslimiter/usagetrackerclient"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/storage"
)

type LimitedStorage struct {
	storage      storage.Appendable
	usageTracker *usagetrackerclient.UsageTrackerClient
	tenant       string
}

var _ storage.Appendable = (*LimitedStorage)(nil)

func NewLimitedStorage(storage storage.Appendable, usageTracker *usagetrackerclient.UsageTrackerClient, tenant string) *LimitedStorage {
	return &LimitedStorage{storage: storage, usageTracker: usageTracker, tenant: tenant}
}

func (l *LimitedStorage) Appender(ctx context.Context) storage.Appender {
	return &limitedAppender{
		appender:     l.storage.Appender(ctx),
		usageTracker: l.usageTracker,
		tenant:       l.tenant,
	}
}

type limitedAppender struct {
	appender     storage.Appender
	usageTracker *usagetrackerclient.UsageTrackerClient
	tenant       string
}

// Append implements storage.Appender.
func (l *limitedAppender) Append(ref storage.SeriesRef, lbls labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	if allow, err := l.allowSeries(lbls); err != nil {
		return 0, err
	} else if !allow {
		return 0, nil
	}

	return l.appender.Append(ref, lbls, t, v)
}

func (l *limitedAppender) allowSeries(lbls labels.Labels) (bool, error) {
	hash := lbls.Hash()
	ctx := user.InjectOrgID(context.Background(), l.tenant)
	rejected, err := l.usageTracker.TrackSeries(ctx, l.tenant, []uint64{hash})
	if err != nil {
		return false, err
	}
	if len(rejected) > 0 {
		// TODO: track metrics here. We can't return an error because the generator treats that as fatal.
		return false, nil
	}
	return true, nil
}

// AppendCTZeroSample implements storage.Appender.
func (l *limitedAppender) AppendCTZeroSample(ref storage.SeriesRef, lbls labels.Labels, t int64, ct int64) (storage.SeriesRef, error) {
	if allow, err := l.allowSeries(lbls); err != nil {
		return 0, err
	} else if !allow {
		return 0, nil
	}
	return l.appender.AppendCTZeroSample(ref, lbls, t, ct)
}

// AppendExemplar implements storage.Appender.
func (l *limitedAppender) AppendExemplar(ref storage.SeriesRef, lbls labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	if allow, err := l.allowSeries(lbls); err != nil {
		return 0, err
	} else if !allow {
		return 0, nil
	}
	return l.appender.AppendExemplar(ref, lbls, e)
}

// AppendHistogram implements storage.Appender.
func (l *limitedAppender) AppendHistogram(ref storage.SeriesRef, lbls labels.Labels, t int64, h *histogram.Histogram, fh *histogram.FloatHistogram) (storage.SeriesRef, error) {
	if allow, err := l.allowSeries(lbls); err != nil {
		return 0, err
	} else if !allow {
		return 0, nil
	}
	return l.appender.AppendHistogram(ref, lbls, t, h, fh)
}

// AppendHistogramCTZeroSample implements storage.Appender.
func (l *limitedAppender) AppendHistogramCTZeroSample(ref storage.SeriesRef, lbls labels.Labels, t int64, ct int64, h *histogram.Histogram, fh *histogram.FloatHistogram) (storage.SeriesRef, error) {
	if allow, err := l.allowSeries(lbls); err != nil {
		return 0, err
	} else if !allow {
		return 0, nil
	}
	return l.appender.AppendHistogramCTZeroSample(ref, lbls, t, ct, h, fh)
}

// Commit implements storage.Appender.
func (l *limitedAppender) Commit() error {
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
	return l.appender.UpdateMetadata(ref, lbls, m)
}

var _ storage.Appender = (*limitedAppender)(nil)
