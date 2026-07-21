// Package bloomgatewayevents holds the producer-side pieces of the bloom
// gateway's write path (DESIGN.md § Write path): config shared by every
// producer (block-builder, backend-worker), and the pure helpers used to
// turn a block's trace IDs into wire-sized, partition-routed chunks. It
// deliberately contains no Kafka client code -- that lands in a later work
// package once this scaffolding is in place.
package bloomgatewayevents

import (
	"errors"
	"flag"
	"fmt"

	"github.com/grafana/tempo/pkg/ingest"
)

const (
	// defaultKafkaTopic must match modules/bloomgateway/config.go's
	// defaultKafkaTopic byte-for-byte: producers and the gateway's consumer
	// only ever meet if they agree on the topic name by construction, not
	// by operator convention.
	defaultKafkaTopic = "tempo.bloom-gateway.events"

	// defaultAutoCreateTopicDefaultPartitions mirrors
	// modules/bloomgateway/config.go's defaultAutoCreateTopicDefaultPartitions
	// (K = 16, DESIGN.md § Write path), overriding ingest.KafkaConfig's own
	// default of 1000 (sized for the block-builder's very different
	// partitioning needs).
	defaultAutoCreateTopicDefaultPartitions = 16

	// defaultChunkSize is DESIGN.md § Write path's reference chunking:
	// "Payloads are chunked at ~200k trace IDs (~3.2 MiB) per message".
	defaultChunkSize = 200_000

	// maxChunkSize bounds ChunkSize so a single AddChunk message stays at
	// or under half of the bloom gateway consumer's own
	// FetchMaxPartitionBytes cap (modules/bloomgateway/consumer.go,
	// consumerFetchMaxPartitionBytes = 8 MiB -- added for the same
	// 2026-07-16 restore-side OOM incident this bound is downstream of),
	// leaving headroom for several chunks to batch through one partition
	// fetch rather than one message alone approaching that ceiling.
	// Derived from defaultChunkSize's own documented ratio (~200k trace
	// IDs / ~3.2 MiB, DESIGN.md § Write path) scaled up to that 4 MiB
	// half-of-cap target: 200_000 * (4 MiB / 3.2 MiB) = 250_000 -- not a
	// hard safety requirement the way the gateway's own decode-time
	// bounds are (bug #8's checkCount, modules/bloomgateway/snapshot.go):
	// franz-go's FetchMaxPartitionBytes still returns an over-cap batch
	// rather than stalling the connection (verified directly against
	// vendor/.../kgo/config.go's own doc comment: "if a single batch is
	// larger than this number, that batch will still be returned so the
	// client can make progress"). The risk this bound actually forecloses
	// is operational: a persistently oversized head-of-partition message
	// producing a standing per-fetch log/alert condition indefinitely,
	// worth catching at config-validation time rather than discovering
	// operationally after the fact.
	maxChunkSize = 250_000
)

// Config is shared by every bloom-gateway event producer (block-builder,
// backend-worker). Each producer embeds this once and calls
// RegisterFlagsAndApplyDefaults under its own prefix.
type Config struct {
	// Enabled gates publishing entirely. Producers publish bloom-gateway
	// events only when enabled; the gateway itself (modules/bloomgateway)
	// is unaffected either way since it only ever reacts to events that
	// arrive.
	Enabled bool `yaml:"enabled"`

	// Kafka is this producer's own client config, mirroring
	// modules/bloomgateway.Config.Kafka: a separate ingest.KafkaConfig
	// instance rather than reusing the ingest-path one, so the topic and
	// partition-count overrides below can never leak into unrelated
	// Kafka usage.
	Kafka ingest.KafkaConfig `yaml:"kafka"`

	// ChunkSize is the number of trace IDs per AddChunk message
	// (DESIGN.md § Write path). Must be in (0, maxChunkSize] -- see that
	// constant's own doc comment for why the upper bound exists.
	ChunkSize int `yaml:"chunk_size"`
}

// RegisterFlagsAndApplyDefaults registers this Config's flags under prefix
// and applies every default documented above. Must be side-effect-free
// beyond mutating cfg: it is called a second time, on a throwaway Config, to
// compute /status/config?mode=defaults|diff (module-wiring report
// convention #9).
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, prefix+".enabled", false, "Enable publishing bloom-gateway events. Producers publish bloom-gateway events only when enabled.")

	cfg.Kafka.RegisterFlagsWithPrefix(prefix+".kafka", f)
	// Same pattern as modules/bloomgateway/config.go's consumer-side
	// override: the two sides must agree on topic and partition count by
	// construction, not by operator convention.
	cfg.Kafka.Topic = defaultKafkaTopic
	cfg.Kafka.AutoCreateTopicDefaultPartitions = defaultAutoCreateTopicDefaultPartitions

	f.IntVar(&cfg.ChunkSize, prefix+".chunk-size", defaultChunkSize, "Number of trace IDs per AddChunk message.")
}

// Validate checks the parts of Config that Go's type system can't. Every
// check is gated on Enabled: a disabled producer's Kafka/chunk-size fields
// are inert and must never block startup, since Enabled defaults to false
// and most deployments will never configure the rest.
func (cfg *Config) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.ChunkSize <= 0 {
		return errors.New("bloom gateway events: chunk_size must be > 0")
	}
	if cfg.ChunkSize > maxChunkSize {
		return fmt.Errorf("bloom gateway events: chunk_size must be <= %d", maxChunkSize)
	}

	if err := cfg.Kafka.Validate(); err != nil {
		return fmt.Errorf("bloom gateway events: kafka: %w", err)
	}

	return nil
}
