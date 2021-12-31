package remotewriteexporter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
)

const (
	sumMetric    = "traces_spanmetrics_latency_sum"
	countMetric  = "traces_spanmetrics_latency_count"
	bucketMetric = "traces_spanmetrics_latency_bucket"
)

func TestRemoteWriteExporter_handleHistogramIntDataPoints(t *testing.T) {
	var (
		countValue     uint64  = 20
		sumValue       float64 = 100
		bucketCounts           = []uint64{1, 2, 3, 4, 5, 6}
		explicitBounds         = []float64{1, 2.5, 5, 7.5, 10}
		ts                     = time.Date(2020, 1, 2, 3, 4, 5, 6, time.UTC)
	)

	appendable := &mockAppendable{}
	exp := remoteWriteExporter{
		namespace: "traces_spanmetrics",
	}
	app := appendable.Appender(context.TODO())

	// Build data point
	dps := pdata.NewHistogramDataPointSlice()
	dp := dps.AppendEmpty()
	dp.SetTimestamp(pdata.NewTimestampFromTime(ts.UTC()))
	dp.SetBucketCounts(bucketCounts)
	dp.SetExplicitBounds(explicitBounds)
	dp.SetCount(countValue)
	dp.SetSum(sumValue)

	err := exp.handleHistogramIntDataPoints(app, "latency", dps)
	require.NoError(t, err)

	// Verify _sum
	sum := appendable.GetAppended(sumMetric)
	require.Equal(t, len(sum), 1)
	require.Equal(t, sum[0].v, sumValue)
	require.Equal(t, sum[0].l, labels.Labels{{Name: nameLabelKey, Value: "traces_spanmetrics_latency_" + sumSuffix}})

	// Check _count
	count := appendable.GetAppended(countMetric)
	require.Equal(t, len(count), 1)
	require.Equal(t, count[0].v, float64(countValue))
	require.Equal(t, count[0].l, labels.Labels{{Name: nameLabelKey, Value: "traces_spanmetrics_latency_" + countSuffix}})

	// Check _bucket
	buckets := appendable.GetAppended(bucketMetric)
	require.Equal(t, len(buckets), len(bucketCounts))
	var bCount uint64
	for i, b := range buckets {
		bCount += bucketCounts[i]
		require.Equal(t, b.v, float64(bCount))
		eb := infBucket
		if len(explicitBounds) > i {
			eb = fmt.Sprint(explicitBounds[i])
		}
		require.Equal(t, b.l, labels.Labels{
			{Name: nameLabelKey, Value: "traces_spanmetrics_latency_" + bucketSuffix},
			{Name: leStr, Value: eb},
		})
	}
}

type mockAppendable struct {
	appender *mockAppender
}

func (m *mockAppendable) Appender(context.Context) storage.Appender {
	if m.appender == nil {
		m.appender = &mockAppender{}
	}
	return m.appender
}

func (m *mockAppendable) GetAppended(metricName string) []metric {
	return m.appender.GetAppended(metricName)
}

type metric struct {
	l labels.Labels
	v float64
}

type mockAppender struct {
	appendedMetrics []metric
}

func (a *mockAppender) GetAppended(n string) []metric {
	var ms []metric
	for _, m := range a.appendedMetrics {
		if n == m.l.Get(nameLabelKey) {
			ms = append(ms, m)
		}
	}
	return ms
}

func (a *mockAppender) Append(_ uint64, l labels.Labels, _ int64, v float64) (uint64, error) {
	a.appendedMetrics = append(a.appendedMetrics, metric{l: l, v: v})
	return 0, nil
}

func (a *mockAppender) Commit() error { return nil }

func (a *mockAppender) Rollback() error { return nil }

func (a *mockAppender) AppendExemplar(_ uint64, _ labels.Labels, _ exemplar.Exemplar) (uint64, error) {
	return 0, nil
}
