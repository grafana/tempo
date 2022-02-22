package remotewrite

import (
	"context"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

// NoopAppender implements storage.Appendable and storage.Appender
type NoopAppender struct{}

var _ storage.Appendable = (*NoopAppender)(nil)
var _ storage.Appender = (*NoopAppender)(nil)

func (a *NoopAppender) Appender(_ context.Context) storage.Appender {
	return a
}

func (a *NoopAppender) Append(_ storage.SeriesRef, _ labels.Labels, _ int64, _ float64) (storage.SeriesRef, error) {
	return 0, nil
}

func (a *NoopAppender) AppendExemplar(_ storage.SeriesRef, _ labels.Labels, _ exemplar.Exemplar) (storage.SeriesRef, error) {
	return 0, nil
}

func (a *NoopAppender) Commit() error {
	return nil
}

func (a *NoopAppender) Rollback() error {
	return nil
}
