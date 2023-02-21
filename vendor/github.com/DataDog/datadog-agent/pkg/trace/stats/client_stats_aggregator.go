// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package stats

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/watchdog"
)

const (
	bucketDuration       = 2 * time.Second
	clientBucketDuration = 10 * time.Second
	oldestBucketStart    = 20 * time.Second
)

const (
	// set on a stat payload containing only distributions post aggregation
	keyDistributions = "distributions"
	// set on a stat payload containing counts (hit/error/duration) post aggregation
	keyCounts = "counts"
)

// ClientStatsAggregator aggregates client stats payloads on buckets of bucketDuration
// If a single payload is received on a bucket, this Aggregator is a passthrough.
// If two or more payloads collide, their counts will be aggregated into one bucket.
// Multiple payloads will be sent:
// - Original payloads with their distributions will be sent with counts zeroed.
// - A single payload with the bucket aggregated counts will be sent.
// This and the aggregator timestamp alignment ensure that all counts will have at most one point per second per agent for a specific granularity.
// While distributions are not tied to the agent.
type ClientStatsAggregator struct {
	In      chan pb.ClientStatsPayload
	out     chan pb.StatsPayload
	buckets map[int64]*bucket // buckets used to aggregate client stats

	flushTicker   *time.Ticker
	oldestTs      time.Time
	agentEnv      string
	agentHostname string
	agentVersion  string

	exit chan struct{}
	done chan struct{}
}

// NewClientStatsAggregator initializes a new aggregator ready to be started
func NewClientStatsAggregator(conf *config.AgentConfig, out chan pb.StatsPayload) *ClientStatsAggregator {
	return &ClientStatsAggregator{
		flushTicker:   time.NewTicker(time.Second),
		In:            make(chan pb.ClientStatsPayload, 10),
		buckets:       make(map[int64]*bucket, 20),
		out:           out,
		agentEnv:      conf.DefaultEnv,
		agentHostname: conf.Hostname,
		agentVersion:  conf.AgentVersion,
		oldestTs:      alignAggTs(time.Now().Add(bucketDuration - oldestBucketStart)),
		exit:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// Start starts the aggregator.
func (a *ClientStatsAggregator) Start() {
	go func() {
		defer watchdog.LogOnPanic()
		for {
			select {
			case t := <-a.flushTicker.C:
				a.flushOnTime(t)
			case input := <-a.In:
				a.add(time.Now(), input)
			case <-a.exit:
				a.flushAll()
				close(a.done)
				return
			}
		}
	}()
}

// Stop stops the aggregator. Calling Stop twice will panic.
func (a *ClientStatsAggregator) Stop() {
	close(a.exit)
	<-a.done
}

// flushOnTime flushes all buckets up to flushTs, except the last one.
func (a *ClientStatsAggregator) flushOnTime(now time.Time) {
	flushTs := alignAggTs(now.Add(bucketDuration - oldestBucketStart))
	for t := a.oldestTs; t.Before(flushTs); t = t.Add(bucketDuration) {
		if b, ok := a.buckets[t.Unix()]; ok {
			a.flush(b.flush())
			delete(a.buckets, t.Unix())
		}
	}
	a.oldestTs = flushTs
}

func (a *ClientStatsAggregator) flushAll() {
	for _, b := range a.buckets {
		a.flush(b.flush())
	}
}

// getAggregationBucketTime returns unix time at which we aggregate the bucket.
// We timeshift payloads older than a.oldestTs to a.oldestTs.
// Payloads in the future are timeshifted to the latest bucket.
func (a *ClientStatsAggregator) getAggregationBucketTime(now, bs time.Time) (time.Time, bool) {
	if bs.Before(a.oldestTs) {
		return a.oldestTs, true
	}
	if bs.After(now) {
		return alignAggTs(now), true
	}
	return alignAggTs(bs), false
}

func (a *ClientStatsAggregator) add(now time.Time, p pb.ClientStatsPayload) {
	for _, clientBucket := range p.Stats {
		clientBucketStart := time.Unix(0, int64(clientBucket.Start))
		ts, shifted := a.getAggregationBucketTime(now, clientBucketStart)
		if shifted {
			clientBucket.AgentTimeShift = ts.Sub(clientBucketStart).Nanoseconds()
			clientBucket.Start = uint64(ts.UnixNano())
		}
		b, ok := a.buckets[ts.Unix()]
		if !ok {
			b = &bucket{ts: ts}
			a.buckets[ts.Unix()] = b
		}
		p.Stats = []pb.ClientStatsBucket{clientBucket}
		a.flush(b.add(p))
	}
}

func (a *ClientStatsAggregator) flush(p []pb.ClientStatsPayload) {
	if len(p) == 0 {
		return
	}
	a.out <- pb.StatsPayload{
		Stats:          p,
		AgentEnv:       a.agentEnv,
		AgentHostname:  a.agentHostname,
		AgentVersion:   a.agentVersion,
		ClientComputed: true,
	}
}

// alignAggTs aligns time to the aggregator timestamps.
// Timestamps from the aggregator are never aligned  with concentrator timestamps.
// This ensures that all counts sent by a same agent host are never on the same second.
// aggregator timestamps:   2ks+1s (1s, 3s, 5s, 7s, 9s, 11s)
// concentrator timestamps: 10ks   (0s, 10s, 20s ..)
func alignAggTs(t time.Time) time.Time {
	return t.Truncate(bucketDuration).Add(time.Second)
}

type bucket struct {
	// first is the first payload matching the bucket. If a second payload matches the bucket
	// this field will be empty
	first pb.ClientStatsPayload
	// ts is the timestamp attached to the payload
	ts time.Time
	// n counts the number of payloads matching the bucket
	n int
	// agg contains the aggregated Hits/Errors/Duration counts
	agg map[PayloadAggregationKey]map[BucketsAggregationKey]*aggregatedCounts
}

func (b *bucket) add(p pb.ClientStatsPayload) []pb.ClientStatsPayload {
	b.n++
	if b.n == 1 {
		b.first = p
		return nil
	}
	// if it's the second payload we flush the first payload with counts trimmed
	if b.n == 2 {
		first := b.first
		b.first = pb.ClientStatsPayload{}
		b.agg = make(map[PayloadAggregationKey]map[BucketsAggregationKey]*aggregatedCounts, 2)
		b.aggregateCounts(first)
		b.aggregateCounts(p)
		return []pb.ClientStatsPayload{trimCounts(first), trimCounts(p)}
	}
	b.aggregateCounts(p)
	return []pb.ClientStatsPayload{trimCounts(p)}
}

func (b *bucket) aggregateCounts(p pb.ClientStatsPayload) {
	payloadAggKey := newPayloadAggregationKey(p.Env, p.Hostname, p.Version, p.ContainerID)
	payloadAgg, ok := b.agg[payloadAggKey]
	if !ok {
		var size int
		for _, s := range p.Stats {
			size += len(s.Stats)
		}
		payloadAgg = make(map[BucketsAggregationKey]*aggregatedCounts, size)
		b.agg[payloadAggKey] = payloadAgg
	}
	for _, s := range p.Stats {
		for _, sb := range s.Stats {
			aggKey := newBucketAggregationKey(sb)
			agg, ok := payloadAgg[aggKey]
			if !ok {
				agg = &aggregatedCounts{}
				payloadAgg[aggKey] = agg
			}
			agg.hits += sb.Hits
			agg.errors += sb.Errors
			agg.duration += sb.Duration
		}
	}
}

func (b *bucket) flush() []pb.ClientStatsPayload {
	if b.n == 1 {
		return []pb.ClientStatsPayload{b.first}
	}
	return b.aggregationToPayloads()
}

func (b *bucket) aggregationToPayloads() []pb.ClientStatsPayload {
	res := make([]pb.ClientStatsPayload, 0, len(b.agg))
	for payloadKey, aggrCounts := range b.agg {
		stats := make([]pb.ClientGroupedStats, 0, len(aggrCounts))
		for aggrKey, counts := range aggrCounts {
			stats = append(stats, pb.ClientGroupedStats{
				Service:        aggrKey.Service,
				Name:           aggrKey.Name,
				Resource:       aggrKey.Resource,
				HTTPStatusCode: aggrKey.StatusCode,
				Type:           aggrKey.Type,
				Synthetics:     aggrKey.Synthetics,
				Hits:           counts.hits,
				Errors:         counts.errors,
				Duration:       counts.duration,
			})
		}
		clientBuckets := []pb.ClientStatsBucket{
			{
				Start:    uint64(b.ts.UnixNano()),
				Duration: uint64(clientBucketDuration.Nanoseconds()),
				Stats:    stats,
			}}
		res = append(res, pb.ClientStatsPayload{
			Hostname:         payloadKey.Hostname,
			Env:              payloadKey.Env,
			Version:          payloadKey.Version,
			Stats:            clientBuckets,
			AgentAggregation: keyCounts,
		})
	}
	return res
}

func newPayloadAggregationKey(env, hostname, version, cid string) PayloadAggregationKey {
	return PayloadAggregationKey{Env: env, Hostname: hostname, Version: version, ContainerID: cid}
}

func newBucketAggregationKey(b pb.ClientGroupedStats) BucketsAggregationKey {
	return BucketsAggregationKey{
		Service:    b.Service,
		Name:       b.Name,
		Resource:   b.Resource,
		Type:       b.Type,
		Synthetics: b.Synthetics,
		StatusCode: b.HTTPStatusCode,
	}
}

func trimCounts(p pb.ClientStatsPayload) pb.ClientStatsPayload {
	p.AgentAggregation = keyDistributions
	for _, s := range p.Stats {
		for i, b := range s.Stats {
			b.Hits = 0
			b.Errors = 0
			b.Duration = 0
			s.Stats[i] = b
		}
	}
	return p
}

// aggregate separately hits, errors, duration
// Distributions and TopLevelCount will stay on the initial payload
type aggregatedCounts struct {
	hits, errors, duration uint64
}
