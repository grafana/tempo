// Forked from https://github.com/grafana/loki/blob/fa6ef0a2caeeb4d31700287e9096e5f2c3c3a0d4/pkg/kafka/partitionring/consumer/client.go

package ingest

import (
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kgo"
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

func (c *Client) Close() {
	close(c.stopCh)  // Signal the monitor goroutine to stop
	c.wg.Wait()      // Wait for the monitor goroutine to exit
	c.Client.Close() // Close the underlying client
}

func NewReaderClientMetrics(component string, reg prometheus.Registerer) *kprom.Metrics {
	return kprom.NewMetrics("tempo_ingest_storage_reader",
		kprom.Registerer(prometheus.WrapRegistererWith(prometheus.Labels{"component": component}, reg)),
		// Do not export the client ID, because we use it to specify options to the backend.
		kprom.FetchAndProduceDetail(kprom.Batches, kprom.Records, kprom.CompressedBytes, kprom.UncompressedBytes))
}
