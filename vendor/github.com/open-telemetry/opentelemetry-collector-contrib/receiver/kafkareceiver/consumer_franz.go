// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"errors"
	"maps"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/metadata"
)

type topicPartition struct {
	topic     string
	partition int32
}

// partitionControl transfers a rewind request from one partition worker to the
// poll loop and reports whether the poll loop applied it.
type partitionControl struct {
	tp      topicPartition
	pc      *pc
	applied chan bool
}

// brokerReadKey is the cache key for OnBrokerRead/OnBrokerWrite metric options.
type brokerReadKey struct {
	nodeID  int32
	outcome string // "success" or "failure"
}

// franzConsumer implements a Kafka consumer using the franz-go client library.
type franzConsumer struct {
	config           *Config
	topics           []string
	excludeTopics    []string
	settings         receiver.Settings
	telemetryBuilder *metadata.TelemetryBuilder
	newConsumeFn     newConsumeMessageFunc
	consumeMessage   consumeMessageFunc

	mu             sync.RWMutex
	started        chan struct{}
	consumerClosed chan struct{}
	closing        chan struct{}

	client      *kgo.Client
	obsrecv     *receiverhelper.ObsReport
	assignments map[topicPartition]*pc
	// controls serializes SetOffsets between PollRecords calls.
	controls chan partitionControl

	// opsMu prevents assignment callbacks and rewinds from racing commits.
	opsMu sync.Mutex

	pollMu     sync.Mutex
	pollCancel context.CancelFunc

	// brokerReadOpts caches MeasurementOptions for OnBrokerRead, which fires on
	// every fetch request. Entries are evicted in OnBrokerDisconnect; growth is
	// bounded by 2 × number-of-brokers (success + failure).
	brokerReadMu   sync.RWMutex
	brokerReadOpts map[brokerReadKey]metric.MeasurementOption

	// ---- status reporting ----
	host         component.Host
	stoppingOnce sync.Once
	stoppedOnce  sync.Once
}

// newFranzKafkaConsumer creates a new franz-go based Kafka consumer
func newFranzKafkaConsumer(
	config *Config,
	set receiver.Settings,
	topics []string,
	excludeTopics []string,
	newConsumeFn newConsumeMessageFunc,
) (*franzConsumer, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &franzConsumer{
		config:           config,
		topics:           topics,
		excludeTopics:    excludeTopics,
		newConsumeFn:     newConsumeFn,
		settings:         set,
		telemetryBuilder: telemetryBuilder,
		started:          make(chan struct{}),
		consumerClosed:   make(chan struct{}),
		closing:          make(chan struct{}),
		assignments:      make(map[topicPartition]*pc),
		controls:         make(chan partitionControl, 1),
		brokerReadOpts:   make(map[brokerReadKey]metric.MeasurementOption),
	}, nil
}

// reportStatus emits a component status event if we have a host.
func (c *franzConsumer) reportStatus(s componentstatus.Status) {
	if c.host == nil {
		return
	}
	componentstatus.ReportStatus(c.host, componentstatus.NewEvent(s))
}

// reportRecoverable reports a recoverable error status event.
func (c *franzConsumer) reportRecoverable(err error) {
	if c.host == nil || err == nil {
		return
	}
	componentstatus.ReportStatus(c.host, componentstatus.NewRecoverableErrorEvent(err))
}

func (c *franzConsumer) Start(ctx context.Context, host component.Host) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.closing:
		return errors.New("franz kafka consumer already shut down")
	case <-c.started:
		return errors.New("franz kafka consumer already started")
	default:
		close(c.started)
	}

	// Report "Starting" as soon as Start() is called.
	c.host = host
	c.reportStatus(componentstatus.StatusStarting)

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             c.settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: c.settings,
	})
	if err != nil {
		return err
	}
	c.obsrecv = obsrecv

	var hooks kgo.Hook = c
	if c.config.Telemetry.Metrics.KafkaReceiverRecordsDelay.Enabled {
		hooks = franzConsumerWithOptionalHooks{c}
	}

	// Create franz-go consumer client
	opts := []kgo.Opt{
		kgo.OnPartitionsAssigned(c.assigned),
		kgo.OnPartitionsRevoked(func(ctx context.Context, _ *kgo.Client, m map[string][]int32) {
			c.lost(ctx, c.client, m, false)
		}),
		kgo.OnPartitionsLost(func(ctx context.Context, _ *kgo.Client, m map[string][]int32) {
			c.lost(ctx, c.client, m, true)
		}),
		kgo.WithHooks(hooks),
	}
	if c.config.PartitionProcessing.Independent {
		// Independent workers and mailboxes are per assignment. Block rebalance
		// callbacks between PollRecords and dispatch so records cannot reach
		// the wrong worker or be dropped. Dispatch does not wait for worker
		// processing. A full mailbox rejects the batch and schedules a rewind.
		// AllowRebalance runs after dispatch, so worker backpressure does not
		// delay a rebalance.
		opts = append(opts, kgo.BlockRebalanceOnPoll())
	}

	if !c.config.ClientConfig.UseLeaderEpoch {
		opts = append(opts, kgo.AdjustFetchOffsetsFn(makeClearLeaderEpochAdjuster()))
	}

	// Create franz-go consumer client
	client, err := kafka.NewFranzConsumerGroup(
		ctx,
		host,
		c.config.ClientConfig,
		c.config.ConsumerConfig,
		c.topics,
		c.excludeTopics,
		c.settings.Logger,
		opts...,
	)
	if err != nil {
		return err
	}
	c.client = client

	cm, err := c.newConsumeFn(host, c.obsrecv, c.telemetryBuilder)
	if err != nil {
		return err
	}
	c.consumeMessage = cm

	go c.consumeLoop(context.Background())
	return nil
}

func (c *franzConsumer) consumeLoop(ctx context.Context) {
	// When the loop exits, report Stopped.
	defer func() {
		c.stoppedOnce.Do(func() { c.reportStatus(componentstatus.StatusStopped) })
		close(c.consumerClosed)
	}()

	for {
		// Consume messages until the ctx is cancelled (the client is closed).
		// Passing -1 drains all records franz-go has buffered; the buffer is
		// bounded by the byte-based fetch limits in ConsumerConfig.
		if !c.consume(ctx, -1) {
			return
		}
	}
}

// consume consumes a batch of messages from the Kafka topic. This is meant to
// be called in a loop until consume returns false.
func (c *franzConsumer) consume(ctx context.Context, size int) bool {
	var fetch kgo.Fetches
	if c.config.PartitionProcessing.Independent {
		c.processControls()
		fetch = c.pollRecords(ctx, size)
		defer c.client.AllowRebalance()
	} else {
		// Poll interruption is only required by independent partition workers.
		fetch = c.client.PollRecords(ctx, size)
	}

	if err := fetch.Err0(); fetch.IsClientClosed() {
		c.settings.Logger.Info("consumer stopped", zap.Error(err))
		return false // Shut down the consumer loop.
	}
	// There's a variety of errors that are returned by fetch.Errors(). We
	// handle the errors that require a client restart above. The rest can
	// simply be logged and keep fetching.
	var hasError bool
	fetch.EachError(func(topic string, partition int32, err error) {
		if c.config.PartitionProcessing.Independent && errors.Is(err, context.Canceled) {
			return
		}
		c.settings.Logger.Error("consumer fetch error", zap.Error(err),
			zap.String("topic", topic),
			zap.Int64("partition", int64(partition)),
		)
		// Report recoverable error while consuming.
		c.reportRecoverable(err)
		hasError = true
	})
	if hasError || fetch.Empty() {
		if c.config.PartitionProcessing.Independent {
			c.processControls()
		}
		return true // Return right away after errors or empty fetch.
	}

	// Acquire the read lock on each consume to ensure the client is not closed
	// and the assignments map is not modified while consuming. Copy the map
	// to avoid locking for the duration of the consume loop.
	c.mu.RLock()
	assignments := maps.Clone(c.assignments)
	c.mu.RUnlock()

	if c.config.PartitionProcessing.Independent {
		c.dispatchPartitionBatches(fetch, assignments)
		c.processControls()
		return true
	}

	var wg sync.WaitGroup
	// Process messages on a per partition basis, wait for them to finish and
	// commit the processed records (if autocommit is disabled).
	fetch.EachPartition(func(p kgo.FetchTopicPartition) {
		count := len(p.Records)
		if count == 0 {
			return // Skip partitions without any records.
		}
		tp := topicPartition{topic: p.Topic, partition: p.Partition}
		assign, ok := assignments[tp]
		// The partition may have been revoked between the time the assignments
		// map was snapshotted above and now.
		if !ok {
			c.settings.Logger.Warn(
				"attempted to process records for a partition not assigned to this consumer",
				zap.String("topic", tp.topic),
				zap.Int64("partition", int64(tp.partition)),
			)
			return
		}
		// Try to add a new in-flight message processing goroutine to the
		// partition consumer. Return immediately if the partition has been
		// lost or reassigned.
		if !assign.add() {
			return
		}
		wg.Add(1)
		assign.logger.Debug("processing fetched records",
			zap.Int("count", count),
			zap.Int64("start_offset", p.Records[0].Offset),
			zap.Int64("end_offset", p.Records[count-1].Offset),
		)
		go func(pc *pc, partition kgo.FetchTopicPartition) {
			defer wg.Done()
			defer pc.wg.Done()
			c.processPartitionBatch(ctx, pc, partition)
		}(assign, p)
	})
	// Wait for all records to be processed and commit if autocommit=false.
	wg.Wait()
	if !c.config.ConsumerConfig.AutoCommit.Enable {
		if err := c.client.CommitMarkedOffsets(ctx); err != nil {
			c.settings.Logger.Error("failed to commit offsets", zap.Error(err))
			// Surface as recoverable error.
			c.reportRecoverable(err)
		}
	}
	return true
}

// pollRecords records the active poll cancellation function so a worker can
// wake the poll loop when an offset operation must run.
func (c *franzConsumer) pollRecords(ctx context.Context, size int) kgo.Fetches {
	pollCtx, cancel := context.WithCancel(ctx)
	c.pollMu.Lock()
	c.pollCancel = cancel
	if len(c.controls) > 0 && c.opsMu.TryLock() {
		c.opsMu.Unlock()
		cancel()
	}
	c.pollMu.Unlock()

	fetch := c.client.PollRecords(pollCtx, size)

	c.pollMu.Lock()
	c.pollCancel = nil
	c.pollMu.Unlock()
	cancel()
	return fetch
}

// sendControl hands an offset operation to the poll loop and reports whether
// it was applied, unless this partition or the receiver is stopping.
func (c *franzConsumer) sendControl(control partitionControl) bool {
	control.applied = make(chan bool, 1)
	select {
	case c.controls <- control:
		// PollRecords may otherwise wait indefinitely while this worker needs
		// the poll loop to apply a rewind.
		c.interruptPoll()
	case <-control.pc.ctx.Done():
		return false
	case <-c.closing:
		return false
	}
	select {
	case applied := <-control.applied:
		return applied
	case <-control.pc.ctx.Done():
		return false
	case <-c.closing:
		return false
	}
}

func (c *franzConsumer) interruptPoll() {
	c.pollMu.Lock()
	defer c.pollMu.Unlock()
	if c.pollCancel != nil {
		c.pollCancel()
	}
}

// unlockOpsAndWakePoll wakes the poll loop if an offset operation arrived
// while another commit, rewind, or assignment callback held opsMu.
func (c *franzConsumer) unlockOpsAndWakePoll() {
	c.opsMu.Unlock()
	if len(c.controls) > 0 {
		c.interruptPoll()
	}
}

// processControls drains all rewind requests before the next PollRecords call.
func (c *franzConsumer) processControls() {
	if !c.opsMu.TryLock() {
		return
	}
	defer c.unlockOpsAndWakePoll()

	for {
		select {
		case control := <-c.controls:
			c.processControlLocked(control)
		default:
			return
		}
	}
}

func (c *franzConsumer) processControlLocked(control partitionControl) {
	applied := false
	defer func() {
		control.applied <- applied
	}()

	c.mu.RLock()
	current := c.assignments[control.tp]
	c.mu.RUnlock()
	if current != control.pc || control.pc.ctx.Err() != nil {
		return
	}

	if offset := control.pc.mailbox.takeOffsetChange(); offset != nil {
		c.client.SetOffsets(map[string]map[int32]kgo.EpochOffset{
			control.tp.topic: {control.tp.partition: *offset},
		})
		applied = true
	}
}

// dispatchPartitionBatches routes each fetched topic-partition to its ordered
// worker without waiting for unrelated partition workers to finish.
func (c *franzConsumer) dispatchPartitionBatches(
	fetch kgo.Fetches,
	assignments map[topicPartition]*pc,
) {
	fetch.EachPartition(func(p kgo.FetchTopicPartition) {
		if len(p.Records) == 0 {
			return
		}
		tp := topicPartition{topic: p.Topic, partition: p.Partition}
		assign, ok := assignments[tp]
		if !ok {
			c.settings.Logger.Warn(
				"attempted to process records for a partition not assigned to this consumer",
				zap.String("topic", tp.topic),
				zap.Int64("partition", int64(tp.partition)),
			)
			return
		}
		partition := map[string][]int32{p.Topic: {p.Partition}}
		enqueued := assign.mailbox.enqueue(p, func(reason partitionPauseReason) {
			assign.addPauseReason(reason)
			c.client.PauseFetchPartitions(partition)
		})
		if !enqueued {
			assign.logger.Debug("discarding fetched records until partition rewind",
				zap.Int64("offset", p.Records[0].Offset),
			)
			return
		}
		assign.logger.Debug("queued fetched records",
			zap.Int("count", len(p.Records)),
			zap.Int64("start_offset", p.Records[0].Offset),
			zap.Int64("end_offset", p.Records[len(p.Records)-1].Offset),
		)
	})
}

func (c *franzConsumer) Shutdown(ctx context.Context) error {
	// Report Stopping at shutdown start.
	c.stoppingOnce.Do(func() { c.reportStatus(componentstatus.StatusStopping) })

	if !c.triggerShutdown() {
		// Idempotent: never fail if not started.
		// We still want to ensure Stopped is eventually emitted (consumeLoop defer handles it).
		// However, if the loop was never started, emit Stopped here too.
		c.stoppedOnce.Do(func() { c.reportStatus(componentstatus.StatusStopped) })
		return nil
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-c.consumerClosed:
	}

	return nil
}

// If it returns false, the caller should return immediately.
func (c *franzConsumer) triggerShutdown() bool {
	c.mu.Lock()
	select {
	case <-c.started:
	default: // Return immediately if the receiver hasn't started.
		c.mu.Unlock()
		return false
	}
	select {
	case <-c.closing:
		c.mu.Unlock()
		return true
	default:
		close(c.closing)
		client := c.client
		c.mu.Unlock()
		// Close the client without holding the write mutex, otherwise, the
		// Shutdown will deadlock when `franzConsumer` inevitably calls the
		// lost/assigned callback.
		if client != nil {
			client.Close()
		}
	}
	return true
}

// assigned must be set as kgo.OnPartitionsAssigned callback. Ensuring all
// assigned partitions to this consumer process received records.
func (c *franzConsumer) assigned(ctx context.Context, cl *kgo.Client, assigned map[string][]int32) {
	c.opsMu.Lock()
	defer c.unlockOpsAndWakePoll()

	// Report OK on each successful assignment so we can recover status after transient errors.
	c.reportStatus(componentstatus.StatusOK)

	// Resume any partitions that were previously paused due to processing errors.
	// PauseFetchPartitions persists across rebalances in franz-go, so we must
	// explicitly resume partitions when they are (re)assigned.
	// ResumeFetchPartitions is a no-op for partitions that are not paused.
	cl.ResumeFetchPartitions(assigned)

	c.mu.Lock()
	defer c.mu.Unlock()
	for topic, partitions := range assigned {
		for _, partition := range partitions {
			c.telemetryBuilder.KafkaReceiverPartitionStart.Add(context.Background(), 1)
			partitionConsumer := pc{
				backOff: newExponentialBackOff(c.config.ErrorBackOff),
				logger: c.settings.Logger.With(
					zap.String("topic", topic),
					zap.Int64("partition", int64(partition)),
				),
				attrs: attribute.NewSet(
					attribute.String("topic", topic),
					attribute.Int64("partition", int64(partition)),
				),
			}
			partitionConsumer.ctx, partitionConsumer.cancel = context.WithCancelCause(ctx)
			tp := topicPartition{topic: topic, partition: partition}
			c.assignments[tp] = &partitionConsumer
			if c.config.PartitionProcessing.Independent {
				partitionConsumer.mailbox = newPartitionMailbox(
					partitionConsumer.ctx,
					c.config.PartitionProcessing.MaxBufferedBatches,
				)
				if partitionConsumer.add() {
					go c.runPartitionWorker(&partitionConsumer, tp)
				}
			}
		}
	}
}

// lost must be set both on kgo.OnPartitionsLost and kgo.OnPartitionsReassigned
// callbacks. Ensures that partitions that are lost (see kgo.OnPartitionsLost
// for more details) or reassigned (see kgo.OnPartitionsReassigned for more
// details) have their partition consumer stopped.
// This callback must finish within the re-balance timeout.
func (c *franzConsumer) lost(ctx context.Context, _ *kgo.Client,
	lost map[string][]int32, fatal bool,
) {
	independent := c.config.PartitionProcessing.Independent

	c.mu.Lock()
	stopping := make(map[topicPartition]*pc)
	for topic, partitions := range lost {
		for _, partition := range partitions {
			tp := topicPartition{topic: topic, partition: partition}
			// In some cases, it is possible for the `lost` to be called with
			// no assignments. So, check if assignments exists first.
			//
			// - OnPartitionLost can be called without the group ever joining
			// and getting assigned.
			// - OnPartitionRevoked can be called multiple times for cooperative
			// balancer on topic lost/deleted.
			pc, ok := c.assignments[tp]
			if !ok {
				continue
			}
			pc.cancelContext(errors.New(
				"stopping processing: partition reassigned or lost",
			))
			// Discard the queue at cancel time. Fatal loss can return while
			// the worker is still inside consumeMessage, and the wait helper
			// keeps pc reachable until that call ends. The in-flight batch
			// stays with the worker.
			if pc.mailbox != nil {
				pc.mailbox.discard()
			}
			stopping[tp] = pc
			if fatal && !independent {
				delete(c.assignments, tp)
			}
			c.telemetryBuilder.KafkaReceiverPartitionClose.Add(context.Background(), 1)
		}
	}
	c.mu.Unlock()

	// Legacy fatal loss returns immediately so the callback stays inside the
	// rebalance timeout. Independent mode has a persistent worker, so wait for
	// it before allowing a replacement worker to start.
	if fatal {
		if independent {
			c.waitForPartitionConsumers(ctx, stopping)
			// Pointer identity prevents this cleanup from deleting a newer
			// assignment if one appeared while the callback was waiting.
			c.deleteStoppingAssignments(stopping)
		}
		return
	}

	for _, pc := range stopping {
		pc.wg.Wait()
	}

	c.opsMu.Lock()
	defer c.unlockOpsAndWakePoll()
	c.deleteStoppingAssignments(stopping)

	// Commit synchronously here (rather than relying on autocommit) so progress
	// is persisted before the partition is reassigned to another consumer,
	// avoiding duplicate processing by the next owner.
	if err := c.client.CommitMarkedOffsets(ctx); err != nil {
		c.settings.Logger.Error("failed to commit offsets", zap.Error(err))
		c.reportRecoverable(err)
	}
}

// waitForPartitionConsumers waits outside the assignment lock so assignment
// state remains available while workers react to cancellation. The timeout
// prevents a downstream consumer that ignores cancellation from blocking the
// Kafka group callback indefinitely.
func (c *franzConsumer) waitForPartitionConsumers(ctx context.Context, stopping map[topicPartition]*pc) {
	workersDone := make(chan struct{})
	go func() {
		defer close(workersDone)
		for _, pc := range stopping {
			pc.wg.Wait()
		}
	}()

	timer := time.NewTimer(c.config.ConsumerConfig.SessionTimeout)
	defer timer.Stop()
	select {
	case <-workersDone:
	case <-ctx.Done():
	case <-timer.C:
		c.settings.Logger.Warn(
			"partition workers did not stop before fatal loss timeout",
			zap.Duration("timeout", c.config.ConsumerConfig.SessionTimeout),
		)
	}
}

// deleteStoppingAssignments removes stopping consumers from c.assignments.
// Delete an entry only when it still points at that consumer, so a newer
// assignment is kept.
func (c *franzConsumer) deleteStoppingAssignments(stopping map[topicPartition]*pc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for tp, pc := range stopping {
		if c.assignments[tp] == pc {
			delete(c.assignments, tp)
		}
	}
}

// handleMessage is called on a per-partition basis.
func (c *franzConsumer) handleMessage(pc *pc, record *kgo.Record) error {
	if pc.backOff != nil {
		defer pc.backOff.Reset()
	}

	for {
		err := c.consumeMessage(pc.ctx, record, pc.attrs)
		if err == nil {
			return nil // Successfully processed.
		}
		// In the future, with Consumer Share Groups, messages not processed
		// within a configurable timeout, are re-delivered to the consumer in
		// the Share group, however, at the time of writing this feature isn't
		// yet GA nor widely deployed.
		// Share groups are generally more in line with observability and OTel
		// collector use cases than traditional consumer groups.
		// https://cwiki.apache.org/confluence/display/KAFKA/KIP-932%3A+Queues+for+Kafka.
		// One possible exception is if the OTel collector is used for analytics
		// pipelines, where it may make sense to make share groups opt-in.
		if pc.backOff != nil && !consumererror.IsPermanent(err) {
			backOffDelay := pc.backOff.NextBackOff()
			if backOffDelay != backoff.Stop {
				pc.logger.Info("Backing off due to error from the next consumer.",
					zap.Error(err),
					zap.Duration("delay", backOffDelay),
				)
				select {
				case <-pc.ctx.Done():
					return context.Cause(pc.ctx)
				case <-time.After(backOffDelay):
					continue
				}
			}
			pc.logger.Warn("Stop error backoff because the configured max_elapsed_time is reached",
				zap.Duration("max_elapsed_time", pc.backOff.MaxElapsedTime),
			)
		}

		isPermanent := consumererror.IsPermanent(err)
		shouldMark := (!isPermanent && c.config.MessageMarking.OnError) || (isPermanent && c.config.MessageMarking.OnPermanentError)

		if c.config.MessageMarking.After && !shouldMark {
			// Only return an error if messages are marked after successful processing.
			return err
		}
		pc.logger.Error("failed to consume message, skipping due to message_marking config",
			zap.Error(err),
			zap.Int64("offset", record.Offset),
		)
		return nil
	}
}

// The methods below implement the relevant franz-go hook interfaces
// record the metrics defined in the metadata telemetry.

// brokerReadOpt returns a cached metric.MeasurementOption for broker read/write hooks.
func (c *franzConsumer) brokerReadOpt(nodeID int32, outcome string) metric.MeasurementOption {
	key := brokerReadKey{nodeID: nodeID, outcome: outcome}
	c.brokerReadMu.RLock()
	opt, ok := c.brokerReadOpts[key]
	c.brokerReadMu.RUnlock()
	if ok {
		return opt
	}
	opt = metric.WithAttributeSet(attribute.NewSet(
		attribute.String("node_id", kgo.NodeName(nodeID)),
		attribute.String("outcome", outcome),
	))
	c.brokerReadMu.Lock()
	c.brokerReadOpts[key] = opt
	c.brokerReadMu.Unlock()
	return opt
}

func (c *franzConsumer) OnBrokerConnect(meta kgo.BrokerMetadata, _ time.Duration, _ net.Conn, err error) {
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	c.telemetryBuilder.KafkaBrokerConnects.Add(
		context.Background(),
		1,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String("node_id", kgo.NodeName(meta.NodeID)),
			attribute.String("outcome", outcome),
		)),
	)
}

func (c *franzConsumer) OnBrokerDisconnect(meta kgo.BrokerMetadata, _ net.Conn) {
	c.telemetryBuilder.KafkaBrokerClosed.Add(
		context.Background(),
		1,
		metric.WithAttributeSet(attribute.NewSet(
			attribute.String("node_id", kgo.NodeName(meta.NodeID)),
		)),
	)
	// Evict cached read opts for this broker.
	c.brokerReadMu.Lock()
	delete(c.brokerReadOpts, brokerReadKey{nodeID: meta.NodeID, outcome: "success"})
	delete(c.brokerReadOpts, brokerReadKey{nodeID: meta.NodeID, outcome: "failure"})
	c.brokerReadMu.Unlock()
}

func (c *franzConsumer) OnBrokerThrottle(meta kgo.BrokerMetadata, throttleInterval time.Duration, _ bool) {
	opt := metric.WithAttributeSet(attribute.NewSet(
		attribute.String("node_id", kgo.NodeName(meta.NodeID)),
	))
	// KafkaBrokerThrottlingDuration is deprecated in favor of KafkaBrokerThrottlingLatency.
	c.telemetryBuilder.KafkaBrokerThrottlingDuration.Record(
		context.Background(),
		throttleInterval.Milliseconds(),
		opt,
	)
	c.telemetryBuilder.KafkaBrokerThrottlingLatency.Record(
		context.Background(),
		throttleInterval.Seconds(),
		opt,
	)
}

func (c *franzConsumer) OnBrokerRead(meta kgo.BrokerMetadata, _ int16, _ int, readWait, timeToRead time.Duration, err error) {
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	opt := c.brokerReadOpt(meta.NodeID, outcome)
	// KafkaReceiverLatency is deprecated in favor of KafkaReceiverReadLatency.
	c.telemetryBuilder.KafkaReceiverLatency.Record(
		context.Background(),
		readWait.Milliseconds()+timeToRead.Milliseconds(),
		opt,
	)
	c.telemetryBuilder.KafkaReceiverReadLatency.Record(
		context.Background(),
		readWait.Seconds()+timeToRead.Seconds(),
		opt,
	)
}

// OnFetchBatchRead is called once per batch read from Kafka.
// https://pkg.go.dev/github.com/twmb/franz-go/pkg/kgo#HookFetchBatchRead
func (c *franzConsumer) OnFetchBatchRead(meta kgo.BrokerMetadata, topic string, partition int32, m kgo.FetchBatchMetrics) {
	opt := metric.WithAttributeSet(attribute.NewSet(
		attribute.String("node_id", kgo.NodeName(meta.NodeID)),
		attribute.String("topic", topic),
		attribute.Int64("partition", int64(partition)),
		attribute.String("compression_codec", compressionFromCodec(m.CompressionType)),
		attribute.String("outcome", "success"),
	))
	// KafkaReceiverMessages is deprecated in favor of KafkaReceiverRecords.
	c.telemetryBuilder.KafkaReceiverMessages.Add(
		context.Background(),
		int64(m.NumRecords),
		opt,
	)
	c.telemetryBuilder.KafkaReceiverRecords.Add(
		context.Background(),
		int64(m.NumRecords),
		opt,
	)
	c.telemetryBuilder.KafkaReceiverBytes.Add(
		context.Background(),
		int64(m.CompressedBytes),
		opt,
	)
	c.telemetryBuilder.KafkaReceiverBytesUncompressed.Add(
		context.Background(),
		int64(m.UncompressedBytes),
		opt,
	)
}

// franzConsumerWithOptionalHooks wraps franzConsumer
// so the optional OnFetchRecordUnbuffered can be enabled.
type franzConsumerWithOptionalHooks struct {
	*franzConsumer
}

// OnFetchRecordUnbuffered is called when a fetched record is unbuffered and ready to be processed.
// Note that this hook may slow down high-volume consuming a bit.
// https://pkg.go.dev/github.com/twmb/franz-go/pkg/kgo#HookFetchRecordUnbuffered
func (c franzConsumerWithOptionalHooks) OnFetchRecordUnbuffered(r *kgo.Record, polled bool) {
	if !polled {
		return // Record metrics when polled by `client.PollRecords()`.
	}
	opt := metric.WithAttributeSet(attribute.NewSet(
		attribute.String("topic", r.Topic),
		attribute.Int64("partition", int64(r.Partition)),
	))
	c.telemetryBuilder.KafkaReceiverRecordsDelay.Record(
		context.Background(),
		time.Since(r.Timestamp).Seconds(),
		opt,
	)
}

func compressionFromCodec(c uint8) string {
	// CompressionType signifies which algorithm the batch was compressed
	// with.
	//
	// 0 is no compression, 1 is gzip, 2 is snappy, 3 is lz4, and 4 is
	// zstd.
	switch c {
	case 0:
		return "none"
	case 1:
		return "gzip"
	case 2:
		return "snappy"
	case 3:
		return "lz4"
	case 4:
		return "zstd"
	default:
		return "unknown"
	}
}

func makeClearLeaderEpochAdjuster() func(context.Context, map[string]map[int32]kgo.Offset) (map[string]map[int32]kgo.Offset, error) {
	return func(_ context.Context, topics map[string]map[int32]kgo.Offset) (map[string]map[int32]kgo.Offset, error) {
		for _, parts := range topics {
			for p, off := range parts {
				parts[p] = off.WithEpoch(-1)
			}
		}
		return topics, nil
	}
}
