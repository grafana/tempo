package generator

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kgo"
)

var metricEnqueueTime = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "tempo",
	Subsystem: "metrics_generator",
	Name:      "enqueue_time_seconds_total",
	Help:      "The total amount of time spent waiting to enqueue for processing",
})

func (g *Generator) startKafka() {
	g.kafkaCh = make(chan *kgo.Record, g.cfg.IngestConcurrency)

	// Create context that will be used to stop the goroutines.
	var ctx context.Context
	ctx, g.kafkaStop = context.WithCancel(context.Background())

	for i := uint(0); i < g.cfg.IngestConcurrency; i++ {
		g.kafkaWG.Add(1)
		go g.readCh(ctx)
	}

	g.kafkaWG.Add(1)
	go g.listenKafka(ctx)
	ingest.ExportPartitionLagMetrics(ctx, g.kafkaAdm, g.logger, g.cfg.Ingest, g.getAssignedActivePartitions, g.kafkaClient.ForceMetadataRefresh)
}

func (g *Generator) stopKafka() {
	g.kafkaStop()
	g.kafkaWG.Wait()
	close(g.kafkaCh)
}

func (g *Generator) listenKafka(ctx context.Context) {
	defer g.kafkaWG.Done()

	level.Info(g.logger).Log("msg", "generator now listening to kafka")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if g.readOnly.Load() {
				// Starting up or shutting down
				continue
			}
			err := g.readKafka(ctx)
			if err != nil {
				level.Error(g.logger).Log("msg", "readKafka failed", "err", err)
				continue
			}
		}
	}
}

func (g *Generator) readKafka(ctx context.Context) error {
	fetches := g.kafkaClient.PollFetches(ctx)
	fetches.EachError(func(_ string, _ int32, err error) {
		if !errors.Is(err, context.Canceled) {
			level.Error(g.logger).Log("msg", "failed to fetch records", "err", err)
		}
	})
	if err := fetches.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Metric lag based on first message in each partition.
	// This balances overhead with granularity.
	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		if len(p.Records) > 0 {
			lag := time.Since(p.Records[0].Timestamp)
			ingest.SetPartitionLagSeconds(g.cfg.Ingest.Kafka.ConsumerGroup, p.Partition, lag)
		}
	})

	start := time.Now()

	for iter := fetches.RecordIter(); !iter.Done(); {
		select {
		case g.kafkaCh <- iter.Next():
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	metricEnqueueTime.Add(time.Since(start).Seconds())

	return nil
}

// readCh reads records from the internal channel.
// This allows for offloading the expensive proto unmarshal
// to multiple goroutines.
func (g *Generator) readCh(ctx context.Context) {
	defer g.kafkaWG.Done()

	var c ingest.GeneratorCodec
	switch g.cfg.Codec {
	case codecPushBytes:
		c = ingest.NewPushBytesDecoder()
	case codecOTLP:
		c = ingest.NewOTLPDecoder()
	}

	for {
		var r *kgo.Record
		select {
		case r = <-g.kafkaCh:
		case <-ctx.Done():
			return
		}

		tenant := string(r.Key)

		i, err := g.getOrCreateInstance(tenant)
		if err != nil {
			level.Error(g.logger).Log("msg", "consumeKafkaChannel getOrCreateInstance", "err", err)
			continue
		}

		iterator, err := c.Decode(r.Value)
		if err != nil {
			level.Error(g.logger).Log("msg", "consumeKafkaChannel decode", "err", err)
			continue
		}

		for resourceSpans, err := range iterator {
			if err != nil {
				level.Error(g.logger).Log("msg", "consumeKafkaChannel unmarshal", "err", err)
				continue
			}

			i.pushSpansFromQueue(ctx, r.Timestamp, resourceSpans)
		}
	}
}

func (g *Generator) getAssignedActivePartitions() []int32 {
	g.partitionMtx.Lock()
	defer g.partitionMtx.Unlock()
	return g.assignedPartitions
}

func (g *Generator) handlePartitionsAssigned(m map[string][]int32) {
	assigned := m[g.cfg.Ingest.Kafka.Topic]
	level.Info(g.logger).Log("msg", "partitions assigned", "partitions", formatInt32Slice(assigned))
	g.partitionMtx.Lock()
	defer g.partitionMtx.Unlock()

	g.assignedPartitions = append(g.assignedPartitions, assigned...)
	sort.Slice(g.assignedPartitions, func(i, j int) bool { return g.assignedPartitions[i] < g.assignedPartitions[j] })
}

func (g *Generator) handlePartitionsRevoked(partitions map[string][]int32) {
	revoked := partitions[g.cfg.Ingest.Kafka.Topic]
	level.Info(g.logger).Log("msg", "partitions revoked", "partitions", formatInt32Slice(revoked))
	g.partitionMtx.Lock()
	defer g.partitionMtx.Unlock()

	sort.Slice(revoked, func(i, j int) bool { return revoked[i] < revoked[j] })
	// Remove revoked partitions
	g.assignedPartitions = revokePartitions(g.assignedPartitions, revoked)

	ingest.ResetLagMetricsForRevokedPartitions(g.cfg.Ingest.Kafka.ConsumerGroup, revoked)
}

// Helper function to format []int32 slice
func formatInt32Slice(slice []int32) string {
	if len(slice) == 0 {
		return "[]"
	}
	result := "["
	for i, v := range slice {
		if i > 0 {
			result += ","
		}
		result += strconv.Itoa(int(v))
	}
	result += "]"
	return result
}

// Helper function to revoke partitions
// Assumes both slices are sorted
func revokePartitions(assigned, revoked []int32) []int32 {
	i, j := 0, 0
	// k is used to track the position where we will overwrite elements in assigned
	k := 0

	// Traverse both slices
	for i < len(assigned) && j < len(revoked) {
		if assigned[i] < revoked[j] {
			// If element in assigned is smaller, it's not in revoked, retain it
			assigned[k] = assigned[i]
			k++
			i++
		} else if assigned[i] > revoked[j] {
			// If element in revoked is smaller, move the pointer j
			j++
		} else {
			// If both elements are equal, skip the element from assigned
			i++
		}
	}

	// If there are leftover elements in assigned, retain them
	for i < len(assigned) {
		assigned[k] = assigned[i]
		k++
		i++
	}

	// Resize assigned to only include retained elements
	return assigned[:k]
}
