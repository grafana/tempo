package generator

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
)

func (g *Generator) startKafka() {
	// Create context that will be used to stop the goroutines.
	var ctx context.Context
	ctx, g.kafkaStop = context.WithCancel(context.Background())

	g.kafkaWG.Add(1)
	go g.listenKafka(ctx)
	ingest.ExportPartitionLagMetrics(ctx, g.kafkaAdm, g.logger, g.cfg.Ingest, g.getAssignedActivePartitions)
}

func (g *Generator) stopKafka() {
	g.kafkaStop()
	g.kafkaWG.Wait()
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
	d := ingest.NewDecoder()

	fetches := g.kafkaClient.PollFetches(ctx)
	fetches.EachError(func(_ string, _ int32, err error) {
		if !errors.Is(err, context.Canceled) {
			level.Error(g.logger).Log("msg", "failed to fetch records", "err", err)
		}
	})

	for iter := fetches.RecordIter(); !iter.Done(); {
		r := iter.Next()

		tenant := string(r.Key)

		i, err := g.getOrCreateInstance(tenant)
		if err != nil {
			return err
		}

		d.Reset()
		req, err := d.Decode(r.Value)
		if err != nil {
			return err
		}

		for _, tr := range req.Traces {
			trace := &tempopb.Trace{}
			err = trace.Unmarshal(tr.Slice)
			if err != nil {
				return err
			}

			i.pushSpansFromQueue(ctx, &tempopb.PushSpansRequest{
				Batches: trace.ResourceSpans,
			})
		}
	}

	return nil
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
