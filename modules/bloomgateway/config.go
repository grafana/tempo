package bloomgateway

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/dskit/flagext"

	"github.com/grafana/tempo/pkg/ingest"
	tempo_ring "github.com/grafana/tempo/pkg/ring"
)

// Defaults. Values mostly come from DESIGN.md's reference sizing (§ Sizing)
// and § Write path; where DESIGN.md doesn't pin a number down, the default
// is a documented, reasonable starting point rather than a value the
// design mandates.
const (
	// defaultD and defaultF are DESIGN.md's reference sizing (§ Sizing):
	// D=25, F=16 -> ~0.9% miss-path false-positive rate at the 100k-block/
	// 20e9-pair reference tenant.
	defaultD uint8 = 25
	defaultF uint8 = 16

	// maxFingerprintBits is the v1 leaf encoding's fixed fingerprint
	// storage width (leaf.go, WP9: parallel fps []uint16 / handles
	// []Handle slices). Widening F past 16 needs a storage-width change,
	// which DESIGN.md already frames as a rare, reshard-like operation (§
	// Changing D, F, or the seed) — this is not a new operational
	// constraint, just an explicit one.
	maxFingerprintBits uint8 = 16

	// maxLeafAddressBits ties D to the ring's token space: a leaf address
	// is routed via LeafRingToken(leafIdx, d) = leafIdx << (32-d) (§ Hash
	// ring), a uint32 ring position, so d must leave room for a real
	// position: d in [1, 32].
	maxLeafAddressBits uint8 = 32

	// maxHashBits is the width of h = xxhash64(trace_id, seed): d and f
	// are disjoint, consecutive slices of it from the top, so d+f must
	// not exceed 64 (§ Sizing: "D + F is the false-positive knob").
	maxHashBits uint8 = 64

	// maxRingTokens is dskit's SpreadMinimizingTokenGenerator hard cap
	// (optimalTokensPerInstance = 512): GenerateTokens silently truncates
	// above it with no error (ring-lifecycler report gotcha #4), so
	// Validate fails fast here instead of leaving an instance silently
	// under-provisioned with no signal.
	maxRingTokens = 512

	// defaultNumTokens matches DESIGN.md's "Tokens are derived
	// deterministically... [dskit's] SpreadMinimizingTokenGenerator" and
	// dskit's own hardcoded optimum.
	defaultNumTokens = maxRingTokens

	// defaultKafkaTopic mirrors DESIGN.md's `tempo.bloom-gateway.events.
	// <cell>` naming (§ Write path), minus the deployment-specific <cell>
	// suffix — multi-cell operators are expected to override this via
	// config, matching how every other topic name in this design is
	// operator-supplied.
	defaultKafkaTopic = "tempo.bloom-gateway.events"

	// defaultAutoCreateTopicDefaultPartitions is K in DESIGN.md § Write
	// path ("K = 16 covers reference sizing"), overriding ingest.
	// KafkaConfig's own default of 1000 (sized for the block-builder's
	// very different partitioning needs).
	defaultAutoCreateTopicDefaultPartitions = 16

	// defaultSnapshotPath is a static default (no per-instance token: in
	// production each instance/pod already has its own isolated
	// filesystem/volume, the same reasoning live-store's WAL path default
	// uses). DESIGN.md's own path format is
	// /var/lib/tempo/bloom-gateway/<instance>/snapshot.bin; the <instance>
	// subdirectory is a deployment/orchestrator concern, not Config's.
	defaultSnapshotPath = "/var/lib/tempo/bloom-gateway/snapshot.bin"

	// defaultSnapshotInterval is DESIGN.md § Snapshots' cadence ("every
	// 4-6h, well inside the 24h topic retention"). Snapshotting defaults
	// ON (paired with defaultSnapshotPath above) because § Availability
	// model's restart-cost story assumes it; "a persistent volume is
	// strongly recommended" (§ Snapshots) but not required for Validate
	// to accept the defaults.
	defaultSnapshotInterval = 5 * time.Hour

	// defaultSweepFullPassPeriod is DESIGN.md § Garbage collection's
	// stated cadence ("full-pass period of ~1-2h").
	defaultSweepFullPassPeriod = 90 * time.Minute

	// defaultTopicRetention/defaultReplayHorizonSlack back
	// defaultSweepReplayHorizon: the sweep must not reclaim a tombstone
	// until its Delete is older than "topic retention + slack" (§ Garbage
	// collection), so resurrection-by-replay stays impossible.
	defaultTopicRetention     = 24 * time.Hour
	defaultReplayHorizonSlack = 30 * time.Minute
	defaultSweepReplayHorizon = defaultTopicRetention + defaultReplayHorizonSlack

	// defaultReconstructionConcurrency is DESIGN.md § Reconstruction's
	// reference fetch concurrency ("~6 min at fetch concurrency 16").
	defaultReconstructionConcurrency = 16

	// defaultReconstructionRateLimitBytesPerSecond is a per-instance slice
	// of DESIGN.md's cell-wide object-store budget for reconstruction/
	// reconciliation reads (§ Scale-in: "a 2-3 GB/s cell budget"; § Sizing
	// reference: 8 instances) — NOT a number DESIGN.md pins down
	// per-instance; documented here as a reasonable starting point subject
	// to tuning, not a mandated value.
	defaultReconstructionRateLimitBytesPerSecond = 256 << 20 // 256 MiB/s

	// defaultReconciliationPeriod is DESIGN.md § Reconciliation's stated
	// cadence ("every few blocklist_poll cycles"; § Operational summary:
	// "every ~15 min").
	defaultReconciliationPeriod = 15 * time.Minute

	// defaultReconciliationLagGate is the consumer-lag threshold above
	// which repair-Adds are suppressed (§ Reconciliation: "lag-gated...
	// skips repair-Adds until lag is back under threshold"). DESIGN.md
	// doesn't pin an exact number down; this default is conservative
	// relative to the reconciliation period above.
	defaultReconciliationLagGate = 5 * time.Minute

	// defaultQueueMaxBytes is DESIGN.md § Sizing's "Event-queue bound +
	// transients ~0.5 GiB" line.
	defaultQueueMaxBytes = 512 << 20 // 512 MiB

	// defaultQueueWorkers is DESIGN.md § Event processing's reference
	// worker-pool size ("A worker pool (16 at reference sizing)").
	defaultQueueWorkers = 16
)

// Config is the top-level bloom gateway configuration.
type Config struct {
	// D is the number of top bits of the trace-ID hash used as the leaf
	// address (leaf count = 2^D). Immutable without a reshard (§ Changing
	// D, F, or the seed).
	D uint8 `yaml:"d"`

	// F is the number of hash bits used as the per-leaf fingerprint width,
	// directly below D's bits. Immutable without a reshard; see
	// maxFingerprintBits for the v1 storage-width restriction.
	F uint8 `yaml:"f"`

	// Seed is the per-cell secret mixed into every trace-ID hash (§ Design
	// constraints). Required; there is deliberately no well-known
	// fallback seed. Shared byte-for-byte across every gateway instance
	// and, later, every query-frontend.
	Seed flagext.Secret `yaml:"seed"`

	// NumTokens is the number of ring tokens this instance registers.
	// DESIGN.md frames this as always dskit's hardcoded optimum (512, §
	// Hash ring); exposed as a field (rather than a hardcoded value)
	// because Validate() and the ring wiring (WP6) both need to reason
	// about it, and because tests need a small value to keep multi-
	// instance ring math legible. Must be in [1, 512].
	NumTokens int `yaml:"num_tokens"`

	Ring  tempo_ring.Config  `yaml:"ring"`
	Kafka ingest.KafkaConfig `yaml:"kafka"`

	Snapshot       SnapshotConfig       `yaml:"snapshot"`
	Sweep          SweepConfig          `yaml:"sweep"`
	Reconstruction ReconstructionConfig `yaml:"reconstruction"`
	Reconciliation ReconciliationConfig `yaml:"reconciliation"`
	Queue          QueueConfig          `yaml:"queue"`
}

// SnapshotConfig configures local-disk snapshotting (§ Snapshots).
type SnapshotConfig struct {
	// Path is the local-disk snapshot file path. Required if Interval > 0.
	Path string `yaml:"path"`

	// Interval is the snapshot cadence. 0 disables snapshotting (every
	// restart becomes a full reconstruction).
	Interval time.Duration `yaml:"interval"`
}

// SweepConfig configures the background garbage-collection sweep (§
// Garbage collection).
type SweepConfig struct {
	// FullPassPeriod is the target wall-clock time for one full pass over
	// the leaf directory.
	FullPassPeriod time.Duration `yaml:"full_pass_period"`

	// ReplayHorizon bounds tombstone reclamation: a deleted block's
	// registry entry is only reclaimed once its Delete is older than this
	// (topic retention + slack, operator-set — not introspected from the
	// broker, §7 invariant #9). Reclaiming earlier would reopen
	// resurrection-by-replay.
	ReplayHorizon time.Duration `yaml:"replay_horizon"`
}

// ReconstructionConfig configures the reconstruction queue (§
// Reconstruction).
type ReconstructionConfig struct {
	// Concurrency bounds parallel object-store column fetches.
	Concurrency int `yaml:"concurrency"`

	// RateLimitBytesPerSecond bounds this instance's share of the
	// cell-wide object-store read budget shared with reconciliation (§
	// Reconstruction, § Reconciliation: "repair fetches share the
	// cell-wide reconstruction rate limit").
	RateLimitBytesPerSecond int64 `yaml:"rate_limit_bytes_per_second"`
}

// ReconciliationConfig configures the periodic tenant-index-vs-registry
// diff loop (§ Reconciliation).
type ReconciliationConfig struct {
	// Period is how often the reconciliation loop runs, per tenant.
	Period time.Duration `yaml:"period"`

	// LagGate is the consumer-lag threshold above which repair-Adds are
	// suppressed (Delete synthesis is unaffected, § Reconciliation).
	LagGate time.Duration `yaml:"lag_gate"`
}

// QueueConfig configures the bounded in-memory event queue between the
// Kafka consumer and the apply worker pool (§ Event processing, §
// Backpressure and memory pressure).
type QueueConfig struct {
	// MaxBytes bounds the queue by bytes, not message count (§
	// Backpressure: "bounded (bytes)").
	MaxBytes int64 `yaml:"max_bytes"`

	// Workers is the fixed size of the apply worker pool.
	Workers int `yaml:"workers"`
}

// RegisterFlagsAndApplyDefaults registers this Config's flags under prefix
// and applies every default documented above. Must be side-effect-free
// beyond mutating cfg: it is called a second time, on a throwaway Config,
// to compute /status/config?mode=defaults|diff (module-wiring report
// convention #9).
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.D = defaultD
	cfg.F = defaultF
	// flag.FlagSet has no native uint8 Var; flag.Func parses+validates the
	// string form directly into the uint8 field without an intermediate
	// int variable (which would need its own separate range check anyway).
	f.Func(prefix+".d", fmt.Sprintf("Number of top trace-ID-hash bits used as the leaf address; leaf count is 2^d. Immutable without a reshard. (default %d)", defaultD), cfg.setD)
	f.Func(prefix+".f", fmt.Sprintf("Number of trace-ID-hash bits used as the per-leaf fingerprint width, max %d. Immutable without a reshard. (default %d)", maxFingerprintBits, defaultF), cfg.setF)

	f.Var(&cfg.Seed, prefix+".seed", "Secret seed mixed into the trace-ID hash. Required; must be identical, byte-for-byte, across every gateway instance (and, later, every query-frontend).")

	f.IntVar(&cfg.NumTokens, prefix+".num-tokens", defaultNumTokens, fmt.Sprintf("Number of tokens this instance registers in the ring. Must be <= %d (dskit SpreadMinimizingTokenGenerator's hard cap).", maxRingTokens))

	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)

	cfg.Kafka.RegisterFlagsWithPrefix(prefix+".kafka", f)
	cfg.Kafka.Topic = defaultKafkaTopic
	cfg.Kafka.AutoCreateTopicDefaultPartitions = defaultAutoCreateTopicDefaultPartitions

	f.StringVar(&cfg.Snapshot.Path, prefix+".snapshot.path", defaultSnapshotPath, "Local-disk path for the snapshot file. Required if snapshot.interval > 0. A persistent volume is strongly recommended (§ Snapshots).")
	f.DurationVar(&cfg.Snapshot.Interval, prefix+".snapshot.interval", defaultSnapshotInterval, "Snapshot cadence. 0 disables snapshotting, making every restart a full reconstruction.")

	f.DurationVar(&cfg.Sweep.FullPassPeriod, prefix+".sweep.full-pass-period", defaultSweepFullPassPeriod, "Target wall-clock time for one full background sweep pass over the leaf directory.")
	f.DurationVar(&cfg.Sweep.ReplayHorizon, prefix+".sweep.replay-horizon", defaultSweepReplayHorizon, "Minimum age of a block's Delete, past the Kafka topic's retention, before its tombstone is reclaimed. Must exceed the topic retention or resurrection-by-replay becomes possible.")

	f.IntVar(&cfg.Reconstruction.Concurrency, prefix+".reconstruction.concurrency", defaultReconstructionConcurrency, "Bounded concurrency for object-store trace-ID column fetches during reconstruction.")
	f.Int64Var(&cfg.Reconstruction.RateLimitBytesPerSecond, prefix+".reconstruction.rate-limit-bytes-per-second", defaultReconstructionRateLimitBytesPerSecond, "This instance's share of the cell-wide object-store read-rate budget, shared between reconstruction and reconciliation repair fetches.")

	f.DurationVar(&cfg.Reconciliation.Period, prefix+".reconciliation.period", defaultReconciliationPeriod, "How often the reconciliation loop diffs the tenant index against the block registry, per tenant.")
	f.DurationVar(&cfg.Reconciliation.LagGate, prefix+".reconciliation.lag-gate", defaultReconciliationLagGate, "Consumer-lag threshold above which reconciliation repair-Adds (not Delete synthesis) are suppressed.")

	f.Int64Var(&cfg.Queue.MaxBytes, prefix+".queue.max-bytes", defaultQueueMaxBytes, "Byte bound on the in-memory queue between the Kafka consumer and the apply worker pool.")
	f.IntVar(&cfg.Queue.Workers, prefix+".queue.workers", defaultQueueWorkers, "Fixed size of the apply worker pool.")
}

// setD and setF back the -<prefix>.d / -<prefix>.f flag.Func registrations.
func (cfg *Config) setD(s string) error {
	v, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid d %q: %w", s, err)
	}
	cfg.D = uint8(v)
	return nil
}

func (cfg *Config) setF(s string) error {
	v, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid f %q: %w", s, err)
	}
	cfg.F = uint8(v)
	return nil
}

// Validate checks the parts of Config that Go's type system can't. It does
// not construct anything (no KV clients, no files opened) — the flag-
// registration-idempotency constraint doesn't apply to Validate directly,
// but keeping it a pure check makes it trivially safe to call repeatedly
// (e.g. from a config-reload path, if one is ever added).
func (cfg *Config) Validate() error {
	if cfg.Seed.String() == "" {
		return errors.New("bloom gateway: seed is required")
	}

	// Order matters for error clarity: d's own bound, then d+f, then f's
	// own bound. With today's constants (d<=32, f<=16) the sum can never
	// exceed 48, so the d+f<=64 check can currently only be reached (and
	// only ever fire) when d's bound already failed to save it — i.e. it
	// is not independently triggerable by an F-only violation today. It
	// is kept anyway, and checked before the f bound, as defense-in-depth
	// against widening maxFingerprintBits later (§ Changing D, F, or the
	// seed's "packed encoding" escape hatch discusses widening F): at that
	// point d+f<=64 becomes the binding constraint again, and it must
	// already be correct rather than newly added under pressure.
	if cfg.D == 0 || cfg.D > maxLeafAddressBits {
		return fmt.Errorf("bloom gateway: d must be between 1 and %d (top bits of the trace-ID hash must address a real position in the ring's 32-bit token space), got %d", maxLeafAddressBits, cfg.D)
	}

	if int(cfg.D)+int(cfg.F) > int(maxHashBits) {
		return fmt.Errorf("bloom gateway: d+f must be <= %d (the trace-ID hash is %d bits wide), got d=%d f=%d (sum %d)", maxHashBits, maxHashBits, cfg.D, cfg.F, int(cfg.D)+int(cfg.F))
	}

	if cfg.F > maxFingerprintBits {
		return fmt.Errorf("bloom gateway: f must be <= %d (the v1 fixed-width leaf encoding's fingerprint storage width), got %d", maxFingerprintBits, cfg.F)
	}

	if cfg.NumTokens <= 0 || cfg.NumTokens > maxRingTokens {
		return fmt.Errorf("bloom gateway: num_tokens must be between 1 and %d, got %d", maxRingTokens, cfg.NumTokens)
	}

	if cfg.Snapshot.Interval > 0 && cfg.Snapshot.Path == "" {
		return errors.New("bloom gateway: snapshot.path is required when snapshot.interval > 0")
	}

	if err := cfg.Kafka.Validate(); err != nil {
		return fmt.Errorf("bloom gateway: kafka: %w", err)
	}

	return nil
}
