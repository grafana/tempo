package remotewriteexporter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/storage"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
	"go.uber.org/atomic"
)

const (
	nameLabelKey  = "__name__"
	sumSuffix     = "sum"
	countSuffix   = "count"
	bucketSuffix  = "bucket"
	leStr         = "le"
	infBucket     = "+Inf"
	counterSuffix = "total"
	noSuffix      = ""
)

type dataPoint interface {
	Attributes() pdata.AttributeMap
}

type remoteWriteExporter struct {
	done       atomic.Bool
	appendable storage.Appendable

	constLabels labels.Labels
	namespace   string

	logger log.Logger
}

func newRemoteWriteExporter(cfg *Config, appendable storage.Appendable) (component.MetricsExporter, error) {
	logger := log.With(util.Logger, "component", "traces remote write exporter")

	ls := make(labels.Labels, 0, len(cfg.ConstLabels))
	for _, constLabel := range cfg.ConstLabels {
		ls = append(ls, labels.Label{
			Name:  constLabel.Name,
			Value: constLabel.Value,
		})
	}

	return &remoteWriteExporter{
		done:        atomic.Bool{},
		appendable:  appendable,
		constLabels: ls,
		logger:      logger,
		namespace:   cfg.Namespace,
	}, nil
}

func (e *remoteWriteExporter) Start(context.Context, component.Host) error { return nil }

func (e *remoteWriteExporter) Shutdown(context.Context) error {
	e.done.Store(true)
	return nil
}

func (e *remoteWriteExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (e *remoteWriteExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	if e.done.Load() {
		return nil
	}

	app := e.appendable.Appender(ctx)

	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		ilm := rm.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilm.Len(); j++ {
			ms := ilm.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				switch m := ms.At(k); m.DataType() {
				case pdata.MetricDataTypeSum, pdata.MetricDataTypeGauge:
					if err := e.processScalarMetric(app, m); err != nil {
						return fmt.Errorf("failed to process metric %s", err)
					}
				case pdata.MetricDataTypeHistogram:
					if err := e.processHistogramMetrics(app, m); err != nil {
						return fmt.Errorf("failed to process metric %s", err)
					}
				case pdata.MetricDataTypeSummary:
					return fmt.Errorf("%s processing unimplemented", m.DataType())
				default:
					return fmt.Errorf("unsupported m data type %s", m.DataType())
				}
			}
		}
	}

	return app.Commit()
}

func (e *remoteWriteExporter) processHistogramMetrics(app storage.Appender, m pdata.Metric) error {
	dps := m.Histogram().DataPoints()
	return e.handleHistogramIntDataPoints(app, m.Name(), dps)
}

func (e *remoteWriteExporter) handleHistogramIntDataPoints(app storage.Appender, name string, dataPoints pdata.HistogramDataPointSlice) error {
	for ix := 0; ix < dataPoints.Len(); ix++ {
		dataPoint := dataPoints.At(ix)
		if err := e.appendDataPoint(app, name, sumSuffix, dataPoint, dataPoint.Sum()); err != nil {
			return err
		}
		if err := e.appendDataPoint(app, name, countSuffix, dataPoint, float64(dataPoint.Count())); err != nil {
			return err
		}

		var cumulativeCount uint64
		for ix, eb := range dataPoint.ExplicitBounds() {
			if ix >= len(dataPoint.BucketCounts()) {
				break
			}
			cumulativeCount += dataPoint.BucketCounts()[ix]
			boundStr := strconv.FormatFloat(eb, 'f', -1, 64)
			ls := labels.Labels{{Name: leStr, Value: boundStr}}
			if err := e.appendDataPointWithLabels(app, name, bucketSuffix, dataPoint, float64(cumulativeCount), ls); err != nil {
				return err
			}
		}
		// add le=+Inf bucket
		cumulativeCount += dataPoint.BucketCounts()[len(dataPoint.BucketCounts())-1]
		ls := labels.Labels{{Name: leStr, Value: infBucket}}
		if err := e.appendDataPointWithLabels(app, name, bucketSuffix, dataPoint, float64(cumulativeCount), ls); err != nil {
			return err
		}

	}
	return nil
}

func (e *remoteWriteExporter) processScalarMetric(app storage.Appender, m pdata.Metric) error {
	switch m.DataType() {
	case pdata.MetricDataTypeSum:
		dataPoints := m.Sum().DataPoints()
		if err := e.handleScalarIntDataPoints(app, m.Name(), counterSuffix, dataPoints); err != nil {
			return err
		}
	case pdata.MetricDataTypeGauge:
		dataPoints := m.Gauge().DataPoints()
		if err := e.handleScalarIntDataPoints(app, m.Name(), noSuffix, dataPoints); err != nil {
			return err
		}
	}
	return nil
}

func (e *remoteWriteExporter) handleScalarIntDataPoints(app storage.Appender, name, suffix string, dataPoints pdata.NumberDataPointSlice) error {
	for ix := 0; ix < dataPoints.Len(); ix++ {
		dataPoint := dataPoints.At(ix)
		if err := e.appendDataPoint(app, name, suffix, dataPoint, float64(dataPoint.IntVal())); err != nil {
			return err
		}
	}
	return nil
}

func (e *remoteWriteExporter) appendDataPoint(app storage.Appender, name, suffix string, dp dataPoint, v float64) error {
	return e.appendDataPointWithLabels(app, name, suffix, dp, v, labels.Labels{})
}

func (e *remoteWriteExporter) appendDataPointWithLabels(app storage.Appender, name, suffix string, dp dataPoint, v float64, customLabels labels.Labels) error {
	ls := e.createLabelSet(name, suffix, dp.Attributes(), customLabels)
	// TODO(mario.rodriguez): Use timestamp from metric
	// time.Now() is used to avoid out-of-order metrics
	ts := timestamp.FromTime(time.Now())
	if _, err := app.Append(0, ls, ts, v); err != nil {
		return err
	}
	return nil
}

func (e *remoteWriteExporter) createLabelSet(name, suffix string, labelMap pdata.AttributeMap, customLabels labels.Labels) labels.Labels {
	ls := make(labels.Labels, 0, labelMap.Len()+1+len(e.constLabels)+len(customLabels))
	// Labels from spanmetrics processor
	labelMap.Range(func(k string, v pdata.AttributeValue) bool {
		ls = append(ls, labels.Label{
			Name:  strings.Replace(k, ".", "_", -1),
			Value: v.StringVal(),
		})
		return true
	})
	// Metric name label
	ls = append(ls, labels.Label{
		Name:  nameLabelKey,
		Value: metricName(e.namespace, name, suffix),
	})
	// Const labels
	ls = append(ls, e.constLabels...)
	// Custom labels
	ls = append(ls, customLabels...)
	return ls
}

func metricName(namespace, metric, suffix string) string {
	if len(suffix) != 0 {
		return fmt.Sprintf("%s_%s_%s", namespace, metric, suffix)
	}
	return fmt.Sprintf("%s_%s", namespace, metric)
}
