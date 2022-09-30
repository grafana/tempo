// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !enable_unstable
// +build !enable_unstable

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"fmt"

	"go.opencensus.io/metric/metricdata"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// queued_retry_inmemory includes the code for memory-backed (original) queued retry helper only
// enabled when "enable_unstable" build tag is not set

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
}

// NewDefaultQueueSettings returns the default settings for QueueSettings.
func NewDefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Enabled:      true,
		NumConsumers: 10,
		// For 5000 queue elements at 100 requests/sec gives about 50 sec of survival of destination outage.
		// This is a pretty decent value for production.
		// User should calculate this from the perspective of how many seconds to buffer in case of a backend outage,
		// multiply that by the number of requests per seconds.
		QueueSize: 5000,
	}
}

// Validate checks if the QueueSettings configuration is valid
func (qCfg *QueueSettings) Validate() error {
	if !qCfg.Enabled {
		return nil
	}

	if qCfg.QueueSize <= 0 {
		return errors.New("queue size must be positive")
	}

	return nil
}

type queuedRetrySender struct {
	fullName        string
	cfg             QueueSettings
	consumerSender  requestSender
	queue           internal.ProducerConsumerQueue
	retryStopCh     chan struct{}
	traceAttributes []attribute.KeyValue
	logger          *zap.Logger
	// currently this is always false for the in-memory queue
	// it's here for consistency with the persistent queue
	requeuingEnabled bool
}

func newQueuedRetrySender(id config.ComponentID, _ config.DataType, qCfg QueueSettings, rCfg RetrySettings, _ internal.RequestUnmarshaler, nextSender requestSender, logger *zap.Logger) *queuedRetrySender {
	retryStopCh := make(chan struct{})
	sampledLogger := createSampledLogger(logger)
	traceAttr := attribute.String(obsmetrics.ExporterKey, id.String())

	qrs := &queuedRetrySender{
		fullName:        id.String(),
		cfg:             qCfg,
		queue:           internal.NewBoundedMemoryQueue(qCfg.QueueSize, func(item interface{}) {}),
		retryStopCh:     retryStopCh,
		traceAttributes: []attribute.KeyValue{traceAttr},
		logger:          sampledLogger,
	}

	qrs.consumerSender = &retrySender{
		traceAttribute:     traceAttr,
		cfg:                rCfg,
		nextSender:         nextSender,
		stopCh:             retryStopCh,
		logger:             sampledLogger,
		onTemporaryFailure: qrs.onTemporaryFailure,
	}

	return qrs
}

func (qrs *queuedRetrySender) onTemporaryFailure(logger *zap.Logger, req request, err error) error {
	if !qrs.requeuingEnabled || qrs.queue == nil {
		logger.Error(
			"Exporting failed. No more retries left. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.count()),
		)
		return err
	}

	if qrs.queue.Produce(req) {
		logger.Error(
			"Exporting failed. Putting back to the end of the queue.",
			zap.Error(err),
		)
	} else {
		logger.Error(
			"Exporting failed. Queue did not accept requeuing request. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.count()),
		)
	}
	return err
}

// start is invoked during service startup.
func (qrs *queuedRetrySender) start(context.Context, component.Host) error {
	qrs.queue.StartConsumers(qrs.cfg.NumConsumers, func(item interface{}) {
		req := item.(request)
		_ = qrs.consumerSender.send(req)
		req.OnProcessingFinished()
	})

	// Start reporting queue length metric
	if qrs.cfg.Enabled {
		err := globalInstruments.queueSize.UpsertEntry(func() int64 {
			return int64(qrs.queue.Size())
		}, metricdata.NewLabelValue(qrs.fullName))
		if err != nil {
			return fmt.Errorf("failed to create retry queue size metric: %w", err)
		}
		err = globalInstruments.queueCapacity.UpsertEntry(func() int64 {
			return int64(qrs.cfg.QueueSize)
		}, metricdata.NewLabelValue(qrs.fullName))
		if err != nil {
			return fmt.Errorf("failed to create retry queue capacity metric: %w", err)
		}
	}

	return nil
}

// shutdown is invoked during service shutdown.
func (qrs *queuedRetrySender) shutdown() {
	// Cleanup queue metrics reporting
	if qrs.cfg.Enabled {
		_ = globalInstruments.queueSize.UpsertEntry(func() int64 {
			return int64(0)
		}, metricdata.NewLabelValue(qrs.fullName))
	}

	// First Stop the retry goroutines, so that unblocks the queue numWorkers.
	close(qrs.retryStopCh)

	// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	if qrs.queue != nil {
		qrs.queue.Stop()
	}
}
