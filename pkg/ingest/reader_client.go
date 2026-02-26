// Forked from https://github.com/grafana/loki/blob/fa6ef0a2caeeb4d31700287e9096e5f2c3c3a0d4/pkg/kafka/partitionring/consumer/client.go

package ingest

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/plugin/kprom"
)

// NewReaderClient returns the kgo.Client that should be used by the Reader.
func NewReaderClient(kafkaCfg KafkaConfig, metrics *kprom.Metrics, logger log.Logger, opts ...kgo.Opt) (*kgo.Client, error) {
	const fetchMaxBytes = 100_000_000

	opts = append(opts, commonKafkaClientOptions(kafkaCfg, metrics, logger)...)
	opts = append(opts,
		kgo.FetchMinBytes(1),
		kgo.FetchMaxBytes(fetchMaxBytes),
		kgo.FetchMaxWait(5*time.Second),
		kgo.FetchMaxPartitionBytes(50_000_000),

		// BrokerMaxReadBytes sets the maximum response size that can be read from
		// Kafka. This is a safety measure to avoid OOMing on invalid responses.
		// franz-go recommendation is to set it 2x FetchMaxBytes.
		kgo.BrokerMaxReadBytes(2*fetchMaxBytes),
	)
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating kafka client")
	}
	if kafkaCfg.AutoCreateTopicEnabled {
		kafkaCfg.SetDefaultNumberOfPartitionsForAutocreatedTopics(logger)
	}
	return client, nil
}

type Client struct {
	logger log.Logger
	*kgo.Client

	wg            sync.WaitGroup
	stopCh        chan struct{}
	partitionRing ring.PartitionRingReader
}

func NewGroupReaderClient(kafkaCfg KafkaConfig, partitionRing ring.PartitionRingReader, metrics *kprom.Metrics, logger log.Logger, opts ...kgo.Opt) (*Client, error) {
	opts = append(opts,
		kgo.ConsumerGroup(kafkaCfg.ConsumerGroup),
		kgo.ConsumeTopics(kafkaCfg.Topic),
		kgo.SessionTimeout(3*time.Minute),
		kgo.RebalanceTimeout(5*time.Minute),
		kgo.Balancers(NewCooperativeActiveStickyBalancer(partitionRing)),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)

	client, err := NewReaderClient(kafkaCfg, metrics, logger, opts...)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Client:        client,
		logger:        logger,
		stopCh:        make(chan struct{}),
		partitionRing: partitionRing,
	}
	// Start the partition monitor goroutine
	c.wg.Add(1)
	go c.monitorPartitions()

	return c, nil
}

func (c *Client) monitorPartitions() {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Get initial partition count from the ring
	lastPartitionCount := c.partitionRing.PartitionRing().PartitionsCount()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// Get current partition count from the ring
			currentPartitionCount := c.partitionRing.PartitionRing().PartitionsCount()
			if currentPartitionCount != lastPartitionCount {
				level.Info(c.logger).Log(
					"msg", "partition count changed, triggering rebalance",
					"previous_count", lastPartitionCount,
					"current_count", currentPartitionCount,
				)
				// Trigger a rebalance to update partition assignments
				// All consumers trigger the rebalance, but only the group leader will actually perform it
				// For non-leader consumers, triggering the rebalance has no effect
				c.ForceRebalance()
				lastPartitionCount = currentPartitionCount
			}
		}
	}
}

// LeaveConsumerGroupByInstanceID sends a LeaveGroup request for the given
// instance ID so the coordinator can rebalance without waiting for session
// timeout. Use this on shutdown when using static membership (InstanceID):
// franz-go does not send LeaveGroup on Close() when InstanceID is set.
// Requires Kafka 2.4+. No-op if instanceID is empty.
func LeaveConsumerGroupByInstanceID(ctx context.Context, client *kgo.Client, group, instanceID string, logger log.Logger) error {
	if instanceID == "" {
		return nil
	}
	req := kmsg.NewPtrLeaveGroupRequest()
	req.Version = 4 // flexible version for Members with InstanceID
	req.Group = group
	member := kmsg.NewLeaveGroupRequestMember()
	member.InstanceID = &instanceID
	req.Members = append(req.Members, member)
	resp, err := req.RequestWith(ctx, client)
	if err != nil {
		return err
	}
	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		return err
	}
	level.Info(logger).Log("msg", "left Kafka consumer group by instance ID", "group", group, "instance_id", instanceID)
	return nil
}

func (c *Client) Close() {
	close(c.stopCh)  // Signal the monitor goroutine to stop
	c.wg.Wait()      // Wait for the monitor goroutine to exit
	c.Client.Close() // Close the underlying client
}

// NewClientForTesting wraps a *kgo.Client in an ingest.Client without starting
// the partition monitor goroutine. For use in unit tests only.
func NewClientForTesting(client *kgo.Client) *Client {
	return &Client{
		Client: client,
		stopCh: make(chan struct{}),
		logger: log.NewNopLogger(),
	}
}

func NewReaderClientMetrics(component string, reg prometheus.Registerer) *kprom.Metrics {
	return kprom.NewMetrics("tempo_ingest_storage_reader",
		kprom.Registerer(prometheus.WrapRegistererWith(prometheus.Labels{"component": component}, reg)),
		// Do not export the client ID, because we use it to specify options to the backend.
		kprom.FetchAndProduceDetail(kprom.Batches, kprom.Records, kprom.CompressedBytes, kprom.UncompressedBytes))
}
