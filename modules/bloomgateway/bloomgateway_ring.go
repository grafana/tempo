package bloomgateway

import (
	"fmt"
	"sort"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"

	tempo_ring "github.com/grafana/tempo/pkg/ring"
)

const (
	// RingName and RingKey identify the gateway's ring both in the KV
	// store and in dskit's own per-ring metric ConstLabels.
	RingName = "bloom-gateway"
	RingKey  = "bloom-gateway"

	// MaxNumTokens is dskit's SpreadMinimizingTokenGenerator hard cap
	// (optimalTokensPerInstance = 512, unexported in vendor/.../ring).
	// GenerateTokens silently truncates above it with no error
	// (ring-lifecycler report gotcha #4) — NewRingManager fails fast
	// instead of leaving an instance silently under-provisioned.
	MaxNumTokens = 512
)

// RingManager owns both halves of the gateway's hash ring: the write side
// (BasicLifecycler, RF=1, deterministic SpreadMinimizingTokenGenerator
// tokens) and the read side (*ring.Ring) query-frontends will eventually
// walk client-side. Bundled into one type so callers manage one object's
// lifecycle instead of hand-wiring the delegate chain themselves (§ Hash
// ring).
type RingManager struct {
	Lifecycler *ring.BasicLifecycler
	Ring       *ring.Ring
}

// NewRingManager builds a RingManager for one gateway instance.
//
// cfg is Tempo's own pkg/ring.Config (the same type WP3's
// bloomgateway.Config.Ring field uses) — its KVStore/HeartbeatPeriod/
// HeartbeatTimeout/instance-address fields drive both the lifecycler and
// the read ring; its ReplicationFactor is not itself a field on this type,
// but cfg.ToRingConfig() (called internally) unconditionally sets RF=1 on
// the dskit ring.Config it returns, which is what actually enforces "the
// gateway runs at replication factor 1" (§ Availability model) — there is
// nothing left for NewRingManager itself to force.
//
// instanceID and zone are threaded through explicitly (rather than read
// off cfg.InstanceID/cfg.InstanceZone) so callers — chiefly this package's
// own multi-instance tests — can share one cfg template across many
// instances while varying only identity. instanceID must match the
// StatefulSet pod-naming shape "name-N" (dskit's
// SpreadMinimizingTokenGenerator requirement); NewRingManager fails fast
// on a mismatch rather than surfacing an unrelated startup failure later
// (ring-lifecycler report gotcha #5).
//
// unregisterOnShutdown and autoForgetTimeout are threaded through as their
// own parameters, not read off cfg, because both are bloom-gateway-
// specific (Config.UnregisterOnShutdown/RingAutoForgetTimeout) and cfg
// here is the type shared with every other ring-backed module in this
// repo (RingStabilityWindow's own doc comment in config.go explains why a
// bloom-gateway-specific field cannot live on tempo_ring.Config itself).
// unregisterOnShutdown sets only this lifecycler's STARTING value of
// KeepInstanceInTheRingOnShutdown (downscale.go's PrepareDownscaleHandler
// flips it at runtime independent of this default, via the exported
// SetKeepInstanceInTheRingOnShutdown).
//
// Deviation from the implementation plan's sketch, noted here because it
// changes the exported signature: the plan's NewRingManager had no way to
// supply the instance's own dial address (ring.BasicLifecyclerConfig.Addr
// is mandatory — other instances/clients need it to connect) or the
// lifecycler's heartbeat PERIOD (as opposed to timeout) as inputs; both
// come for free from cfg via cfg.ToLifecyclerConfig, which is why cfg is
// pkg/ring.Config rather than the bare dskit ring.Config the plan's sketch
// otherwise reads as intending (dskit's ring.Config has neither instance
// identity nor an address/interface-resolution story).
func NewRingManager(cfg tempo_ring.Config, instanceID, zone string, numTokens int, unregisterOnShutdown bool, autoForgetTimeout time.Duration, logger log.Logger, reg prometheus.Registerer) (*RingManager, error) {
	if cfg.HeartbeatTimeout <= 0 {
		return nil, fmt.Errorf("bloom gateway ring: heartbeat timeout must be greater than 0, got %s", cfg.HeartbeatTimeout)
	}
	if numTokens <= 0 || numTokens > MaxNumTokens {
		return nil, fmt.Errorf("bloom gateway ring: num tokens must be between 1 and %d, got %d", MaxNumTokens, numTokens)
	}

	cfg.InstanceID = instanceID
	cfg.InstanceZone = zone

	// Constructed before anything else so its instanceID-shape parse
	// error (the "name-N" regex) surfaces immediately and unambiguously,
	// rather than failing later inside NewBasicLifecycler with a less
	// specific error (DESIGN.md open item #2 / ring-lifecycler report
	// recommendation #3). Single-zone: DESIGN.md never calls for
	// multi-zone bloom-gateway deployments, and a single-element
	// spreadMinimizingZones slice containing exactly zone (even the empty
	// string) is what NewSpreadMinimizingTokenGenerator requires for a
	// single-zone deployment (ring-lifecycler report gotcha #5) —
	// canJoinEnabled=false per §0 D8 / AMENDMENT-adjacent decision:
	// tokens are a pure function of ordinal, not of peer registration
	// order, and DESIGN.md's availability model already tolerates
	// transient ring disagreement.
	tokenGenerator, err := ring.NewSpreadMinimizingTokenGenerator(instanceID, zone, []string{zone}, false)
	if err != nil {
		return nil, fmt.Errorf("bloom gateway ring: instance ID %q: %w (instance IDs must match \"name-N\", e.g. a StatefulSet pod name, for deterministic token generation)", instanceID, err)
	}

	lifecyclerCfg, err := cfg.ToLifecyclerConfig(numTokens)
	if err != nil {
		return nil, fmt.Errorf("bloom gateway ring: resolving instance address: %w", err)
	}
	// ToLifecyclerConfig (pkg/ring/config.go) does not set
	// HeartbeatTimeout or RingTokenGenerator on the BasicLifecyclerConfig
	// it returns — both are required here and would otherwise silently
	// default to zero/RandomTokenGenerator.
	lifecyclerCfg.HeartbeatTimeout = cfg.HeartbeatTimeout
	lifecyclerCfg.RingTokenGenerator = tokenGenerator
	// KeepInstanceInTheRingOnShutdown's STARTING value (2026-07-16
	// shutdown-semantics redesign, DESIGN.md § Availability model
	// amendment) — !unregisterOnShutdown, so this defaults to true (keep)
	// unless the operator has asked for the pre-redesign behavior
	// unconditionally. downscale.go's PrepareDownscaleHandler flips the
	// lifecycler's own runtime copy of this (SetKeepInstanceInTheRingOnShutdown)
	// independent of whatever this static default was, for one intentional
	// removal — this field only matters for a process that never sees a
	// prepare-downscale POST at all.
	lifecyclerCfg.KeepInstanceInTheRingOnShutdown = !unregisterOnShutdown

	// Every existing Tempo ring re-wraps with this prefix before creating
	// dskit ring/lifecycler metrics (module-wiring report convention); it
	// is what makes the vendored ring's own "ring_members" etc. series
	// show up as "tempo_ring_members{name=\"bloom-gateway\",...}".
	reg = prometheus.WrapRegistererWithPrefix("tempo_", reg)

	store, err := kv.NewClient(cfg.KVStore, ring.GetCodec(), kv.RegistererWithKVName(reg, RingName+"-lifecycler"), logger)
	if err != nil {
		return nil, fmt.Errorf("bloom gateway ring: creating KV client: %w", err)
	}

	// Delegates are chained innermost-first (backend-worker's convention:
	// "in reverse order... because they're chained via 'next delegate'").
	// ring.NewInstanceRegisterDelegate is the ONLY delegate in the
	// vendored ring package whose OnRingInstanceRegister actually calls
	// lifecycler.GetTokenGenerator() — every hand-rolled delegate
	// elsewhere in this repo (backend-worker, distributor) ignores
	// RingTokenGenerator entirely and generates tokens its own way; do
	// not copy those as a starting point (ring-lifecycler report gotcha
	// #2 — this is the single most important gotcha for this file).
	//
	// ring.NewLeaveOnStoppingDelegate is deliberately NOT used here, and
	// (2026-07-16 shutdown-semantics redesign, second pass) NO delegate is
	// substituted in its place at all -- reported prominently per the
	// harness's own instructions, since the first pass got this wrong. That
	// first pass tried a conditionalLeaveOnStoppingDelegate wrapper, gating
	// a changeState(LEAVING) call on KeepInstanceInTheRingOnShutdown; it was
	// provably dead code. dskit's services.Service guarantees RunningFn
	// happens-before StoppingFn (vendor/.../dskit/services/basic_service.go),
	// so by the time OnRingInstanceStopping runs, BasicLifecycler.running()
	// -- the ONLY reader of the lifecycler's actorChan -- has already
	// exited. Any call reaching the exported ChangeState (which goes
	// through run() -> actorChan, basic_lifecycler.go) therefore ALWAYS
	// returns "lifecycler not running" at that point; only the vendored
	// delegate's own UNEXPORTED changeState (a direct CAS, no actorChan
	// involved) can actually reach LEAVING from inside stopping(), and this
	// package, outside the ring package, cannot call it. Confirmed
	// empirically: every prepared stop logged that failure and unregistered
	// anyway.
	//
	// No delegate is needed here at all, it turns out: BasicLifecycler.
	// stopping() (basic_lifecycler.go) reads ShouldKeepInstanceInTheRingOn
	// Shutdown() DIRECTLY -- independent of whatever the delegate's own
	// OnRingInstanceStopping does -- to decide keep-vs-unregister; that is
	// the actual mechanism this file's KeepInstanceInTheRingOnShutdown
	// wiring above relies on, and it works with zero delegate involvement.
	// The LEAVING stopover itself was never load-bearing for US
	// specifically, either: this package's own ringOp={ACTIVE}
	// (bloomgateway.go) treats LEAVING identically to absent, so "briefly
	// LEAVING then removed" and "removed directly" are indistinguishable to
	// GetAllHealthy. Omitting a stopping delegate entirely --
	// InstanceRegisterDelegate's own OnRingInstanceStopping is already a
	// no-op -- is therefore strictly simpler than, and (unlike its
	// predecessor) actually correct in achieving, what a conditional
	// wrapper was trying to do.
	delegate := ring.BasicLifecyclerDelegate(ring.NewInstanceRegisterDelegate(ring.ACTIVE, numTokens))
	delegate = ring.NewAutoForgetDelegate(autoForgetTimeout, delegate, logger)

	lifecycler, err := ring.NewBasicLifecycler(lifecyclerCfg, RingName, RingKey, store, delegate, logger, reg)
	if err != nil {
		return nil, fmt.Errorf("bloom gateway ring: creating lifecycler: %w", err)
	}

	// ToRingConfig unconditionally sets ReplicationFactor=1 and
	// SubringCacheDisabled=true (pkg/ring/config.go) — this is the actual
	// enforcement point for "the gateway runs at replication factor 1".
	r, err := ring.New(cfg.ToRingConfig(), RingName, RingKey, logger, reg)
	if err != nil {
		return nil, fmt.Errorf("bloom gateway ring: creating ring: %w", err)
	}

	return &RingManager{Lifecycler: lifecycler, Ring: r}, nil
}

// Services returns the sub-services the caller's services.Manager must
// start and watch together (lifecycler before ring, matching every other
// Tempo ring's starting() sequence).
func (rm *RingManager) Services() []services.Service {
	return []services.Service{rm.Lifecycler, rm.Ring}
}

// LeafRingToken maps a leaf index to its single ring position (§ Hash
// ring: "A leaf maps to the single ring position leaf_idx << (32 - D)").
// Precondition: d <= 32 (bloomgateway.Config.Validate enforces this); for
// d > 32, 32-d underflows uint8 arithmetic into a huge shift count, which
// Go's unbounded-shift semantics turn into "always returns 0" rather than
// a panic — silently wrong, not unsafe, but callers must not rely on this
// function to validate d for them.
func LeafRingToken(leafIdx uint32, d uint8) uint32 {
	return leafIdx << (32 - d)
}

// LeafRange is a contiguous, half-open span of leaf indices: [Start, End).
type LeafRange struct {
	Start, End uint32
}

// OwnedLeafRanges walks instances once and returns the coalesced leaf-index
// ranges owned by instanceID — the reconstruction queue's unit of work (§
// Reconstruction: "coalesces all pending ranges... into a single column
// pass").
//
// Deviation from the implementation plan's sketch:
// OwnedLeafRanges(desc ring.ReplicationSet, ringDesc *ring.Desc,
// instanceID string, d uint8) []LeafRange. The *ring.Desc parameter is
// dropped: a ring.ReplicationSet's Instances already carry each instance's
// Id and Tokens (verified directly against vendor/.../dskit/ring/ring.go's
// GetAllHealthy/GetReplicationSetForOperation, which copy whole
// InstanceDesc values, Tokens included, into the ReplicationSet they
// return) — a *ring.Desc added nothing this function needed and would only
// have made it harder to unit test (ring.Desc's useful accessors for this
// purpose, e.g. getTokensInfo, are unexported). Taking []ring.InstanceDesc
// directly (rather than a ring.ReplicationSet wrapper) lets callers pass
// replicationSet.Instances directly and lets tests build the input by hand
// with no dependency on unexported ring internals.
//
// Ownership semantics deliberately mirror dskit's OWN token-ownership rule
// exactly (vendor/.../dskit/ring/util.go's searchToken, exercised via
// Ring.Get/findInstancesForKey): the owner of ring position p is the
// instance holding the smallest token strictly greater than p, wrapping
// around to the smallest token overall if p is at or past the largest
// token. This is intentionally independent of instance State — matching
// findInstancesForKey's own behavior at RF=1 with no
// shouldExtendReplicaSet function (this package's ringOp,
// ring.NewOp([]ring.InstanceState{ring.ACTIVE}, nil), never extends), the
// walk finds the next token regardless of state and leaves it to the
// caller/ring health machinery to decide whether that instance is
// currently fit to serve — OwnedLeafRanges answers "whose range is this
// structurally", the same question the reconstruction queue needs
// answered to decide what to backfill, independent of transient health.
func OwnedLeafRanges(instances []ring.InstanceDesc, instanceID string, d uint8) []LeafRange {
	type token struct {
		value uint32
		id    string
	}

	var tokens []token
	for _, inst := range instances {
		for _, t := range inst.Tokens {
			tokens = append(tokens, token{value: t, id: inst.Id})
		}
	}
	if len(tokens) == 0 {
		return nil
	}

	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].value != tokens[j].value {
			return tokens[i].value < tokens[j].value
		}
		// Real rings never assign the same token to two instances (every
		// token generator in vendor/.../dskit/ring de-duplicates against
		// allTakenTokens at registration time); this tie-break exists
		// only so a hand-built, deliberately-inconsistent test input has
		// a deterministic (if arbitrary) answer instead of a flaky one.
		return tokens[i].id < tokens[j].id
	})

	numLeaves := uint64(1) << d
	step := uint64(1) << (32 - uint64(d)) // ring positions per leaf

	// leafCeil returns the smallest leaf index k with k*step >= p — i.e.
	// converts a ring-position bound into a leaf-index bound.
	leafCeil := func(p uint64) uint64 {
		return (p + step - 1) / step
	}

	var owned []LeafRange
	addArc := func(ownerID string, start, end uint64) {
		if ownerID != instanceID || start >= end {
			return
		}
		lo, hi := leafCeil(start), leafCeil(end)
		if hi > numLeaves {
			hi = numLeaves // defensive; end <= 2^32 == numLeaves*step always holds
		}
		if lo < hi {
			owned = append(owned, LeafRange{Start: uint32(lo), End: uint32(hi)})
		}
	}

	const ringSize = uint64(1) << 32

	// Token i's arc is [tokens[i-1].value, tokens[i].value) — inclusive of
	// its predecessor's token, exclusive of its own — exactly mirroring
	// searchToken's "first token > key, wrapping to index 0" rule: a
	// position equal to a token belongs to the NEXT token's owner, not
	// that token's own owner.
	addArc(tokens[0].id, 0, uint64(tokens[0].value))
	for i := 1; i < len(tokens); i++ {
		addArc(tokens[i].id, uint64(tokens[i-1].value), uint64(tokens[i].value))
	}
	// The wrap-around arc from the largest token to the end of the ring
	// belongs to the smallest token's owner (the "next" token when
	// walking off the end wraps to index 0).
	addArc(tokens[0].id, uint64(tokens[len(tokens)-1].value), ringSize)

	if len(owned) == 0 {
		return nil
	}

	sort.Slice(owned, func(i, j int) bool { return owned[i].Start < owned[j].Start })

	coalesced := owned[:1]
	for _, r := range owned[1:] {
		last := &coalesced[len(coalesced)-1]
		if r.Start <= last.End {
			if r.End > last.End {
				last.End = r.End
			}
			continue
		}
		coalesced = append(coalesced, r)
	}
	return coalesced
}
