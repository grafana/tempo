// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/metadata"
)

// partitionTelemetry records completed waits and active calls for one assignment.
type partitionTelemetry struct {
	builder        *metadata.TelemetryBuilder
	options        [1]metric.AddOption
	decodeOptions  [1]metric.RecordOption
	consumeOptions [1]metric.RecordOption
	mu             sync.Mutex
	waiting        bool
	active         int64
	stopped        bool
	recovering     bool
	waitStart      time.Time
}

// newPartitionTelemetry creates telemetry with cached stage attributes.
func newPartitionTelemetry(builder *metadata.TelemetryBuilder, attrs attribute.Set) *partitionTelemetry {
	stageOptions := func(stage string) [1]metric.RecordOption {
		values := attrs.ToSlice()
		values = append(values, attribute.String("stage", stage))
		return [1]metric.RecordOption{metric.WithAttributeSet(attribute.NewSet(values...))}
	}
	options := [1]metric.AddOption{metric.WithAttributeSet(attrs)}
	builder.KafkaReceiverPartitionDataWait.Add(context.Background(), 0, options[:]...)
	builder.KafkaReceiverPartitionInFlightRecords.Add(context.Background(), 0, options[:]...)
	return &partitionTelemetry{
		builder:        builder,
		options:        options,
		decodeOptions:  stageOptions("decode"),
		consumeOptions: stageOptions("consume"),
	}
}

// setWaiting updates whether the worker is waiting for data.
func (p *partitionTelemetry) setWaiting(waiting bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waiting = waiting
	p.updateWait()
}

// startCall starts tracking a record-processing call.
func (p *partitionTelemetry) startCall() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	p.active++
	p.builder.KafkaReceiverPartitionInFlightRecords.Add(context.Background(), 1, p.options[:]...)
	p.updateWait()
}

// finishCall completes a call and enters recovery on failure.
func (p *partitionTelemetry) finishCall(err error) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	p.active--
	p.builder.KafkaReceiverPartitionInFlightRecords.Add(context.Background(), -1, p.options[:]...)
	// A failed cross-fetch call can finish while the worker awaits data.
	// Keep recovery separate from waiting until the rewind has completed.
	if err != nil {
		p.recovering = true
	}
	p.updateWait()
}

// resume clears recovery after a rewind.
func (p *partitionTelemetry) resume() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recovering = false
	p.updateWait()
}

// stop ends the wait and clears active calls for this assignment.
func (p *partitionTelemetry) stop() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	p.stopped = true
	p.updateWait()
	p.builder.KafkaReceiverPartitionInFlightRecords.Add(context.Background(), -p.active, p.options[:]...)
	p.active = 0
}

// updateWait runs under mu. A cross-fetch worker can be waiting for another
// batch while earlier calls run; only the last call's return starts data wait.
func (p *partitionTelemetry) updateWait() {
	waiting := p.waiting && p.active == 0 && !p.stopped && !p.recovering
	if waiting && p.waitStart.IsZero() {
		p.waitStart = time.Now()
	} else if !waiting && !p.waitStart.IsZero() {
		p.builder.KafkaReceiverPartitionDataWait.Add(context.Background(), time.Since(p.waitStart).Seconds(), p.options[:]...)
		p.waitStart = time.Time{}
	}
}
