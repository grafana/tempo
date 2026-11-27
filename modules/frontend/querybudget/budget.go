// Package querybudget limits the read work a tenant can start in a query-frontend with a token bucket per tenant.
// The bucket is measured in units of about one estimated byte read. Queries are only admitted while the bucket
// of each of their tenants is not in debt. Jobs are charged with their estimated cost when they are sent to the
// queriers, and block jobs are corrected with the bytes they actually read once they complete.
package querybudget

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
)

// Modes of the query budget of a tenant.
const (
	// ModeDisabled turns the query budget off. Nothing is tracked.
	ModeDisabled = ""
	// ModeDryRun tracks the budget and counts the queries that would be rejected, but rejects none.
	ModeDryRun = "dry_run"
	// ModeEnabled rejects queries of tenants whose bucket is in debt.
	ModeEnabled = "enabled"
)

const (
	// cleanupInterval is how often idle tenants are removed, and idleTimeout how long a tenant has to be idle.
	cleanupInterval = time.Minute
	idleTimeout     = 15 * time.Minute

	// minRetryAfter is the shortest retry delay returned for a rejected query.
	minRetryAfter = time.Second

	labelTenant = "tenant"
)

var (
	levelDesc = prometheus.NewDesc(
		"tempo_query_frontend_query_budget_level",
		"Units left in the query budget bucket of a tenant. Negative values are debt that has to refill before queries are admitted again.",
		[]string{labelTenant}, nil,
	)
)

// Limits are the per-tenant limits of the query budget.
type Limits interface {
	QueryBudgetMode(tenant string) string
	QueryBudgetRate(tenant string) uint64
	QueryBudgetBurst(tenant string) uint64
}

// Outcome is how the query budget decided on a query.
type Outcome int

const (
	// OutcomeAdmitted admits the query. It is the zero value, also used if none of the tenants has a query budget.
	OutcomeAdmitted Outcome = iota
	// OutcomeRejected rejects the query because the bucket of a tenant in ModeEnabled is in debt. The query must
	// not start.
	OutcomeRejected
	// OutcomeDryRunRejected admits the query, but a tenant in ModeDryRun would have rejected it.
	OutcomeDryRunRejected
)

// String returns the name of o as it is logged.
func (o Outcome) String() string {
	switch o {
	case OutcomeAdmitted:
		return "admitted"
	case OutcomeRejected:
		return "rejected"
	case OutcomeDryRunRejected:
		return "dry_run_rejected"
	default:
		return "unknown"
	}
}

// Decision is the result of admitting a query.
type Decision struct {
	Outcome Outcome
	// RetryAfter is the time until the buckets that rejected the query are out of debt again. It is only set
	// for OutcomeRejected.
	RetryAfter time.Duration
	// Query charges the jobs of an admitted query. It is nil if the query is rejected or none of its tenants
	// has a query budget.
	Query *Query
}

// queryTenant is the state of one tenant of a query.
type queryTenant struct {
	// dryRunRejected is set if the tenant is in ModeDryRun and would have rejected the query. The jobs of the
	// tenant are then counted but don't change its bucket, so that the bucket matches that of ModeEnabled.
	dryRunRejected bool

	// forgiven are the units charged to the tenant beyond its debt cap. Later corrections of the query first
	// take them back. It is guarded by the mutex of the Budget.
	forgiven float64
}

// Budget tracks the query budget of all tenants of a query-frontend. It is safe for concurrent use.
type Budget struct {
	limits Limits
	now    func() time.Time

	mtx         sync.Mutex
	buckets     map[string]*bucket
	lastCleanup time.Time

	rejected *prometheus.CounterVec
	charged  *prometheus.CounterVec
	refunded *prometheus.CounterVec
}

// New returns a Budget with the limits of each tenant. Without limits every tenant is disabled.
func New(limits Limits, reg prometheus.Registerer) *Budget {
	factory := promauto.With(reg)
	b := &Budget{
		limits:  limits,
		now:     time.Now,
		buckets: map[string]*bucket{},
		rejected: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "tempo_query_frontend_query_budget_rejected_queries_total",
			Help: "Queries rejected because the query budget of a tenant was in debt. With mode dry_run the queries were admitted anyway.",
		}, []string{labelTenant, "op", "mode"}),
		charged: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "tempo_query_frontend_query_budget_charged_total",
			Help: "Query budget units charged to a tenant: the estimates of the jobs it sent to the queriers and the bytes block jobs read above their estimate.",
		}, []string{labelTenant, "op"}),
		refunded: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "tempo_query_frontend_query_budget_refunded_total",
			Help: "Query budget units given back to a tenant: the bytes block jobs read below their estimate and the estimates of jobs that failed before a querier took them.",
		}, []string{labelTenant, "op"}),
	}
	if reg != nil {
		reg.MustRegister(b)
	}
	return b
}

// Admit decides whether a query of op for tenants may start. A query is rejected if the bucket of a tenant in
// ModeEnabled is in debt. Tenants in ModeDisabled are ignored, and tenants without a rate are only counted.
func (b *Budget) Admit(tenants []string, op jobcost.Op) Decision {
	if b.limits == nil {
		return Decision{}
	}

	now := b.now()
	b.cleanup(now)

	var (
		d        Decision
		budgeted = make(map[string]*queryTenant, len(tenants))
	)
	for _, tenant := range tenants {
		mode := b.limits.QueryBudgetMode(tenant)
		if mode == ModeDisabled {
			continue
		}

		qt := &queryTenant{}
		budgeted[tenant] = qt
		rate, burst := b.rateAndBurst(tenant)

		b.mtx.Lock()
		level := b.useBucketLocked(tenant, now, rate, burst).level
		b.mtx.Unlock()

		if rate == 0 || level >= 0 {
			continue
		}

		if mode == ModeEnabled {
			b.rejected.WithLabelValues(tenant, op.String(), mode).Inc()
			d.Outcome = OutcomeRejected
			d.RetryAfter = max(d.RetryAfter, retryAfter(level, rate))
		} else {
			qt.dryRunRejected = true
		}
	}

	if d.Outcome == OutcomeRejected {
		return d
	}

	// dry run rejections are only counted for queries that are admitted, which is what dry run reports
	for tenant, qt := range budgeted {
		if qt.dryRunRejected {
			b.rejected.WithLabelValues(tenant, op.String(), ModeDryRun).Inc()
			d.Outcome = OutcomeDryRunRejected
		}
	}
	if len(budgeted) > 0 {
		d.Query = &Query{budget: b, op: op.String(), tenants: budgeted}
	}
	return d
}

// Describe implements prometheus.Collector.
func (b *Budget) Describe(ch chan<- *prometheus.Desc) {
	ch <- levelDesc
}

// Collect implements prometheus.Collector. It reports the level of every bucket with a rate.
func (b *Budget) Collect(ch chan<- prometheus.Metric) {
	if b.limits == nil {
		return
	}

	now := b.now()
	levels := map[string]float64{}

	b.mtx.Lock()
	for tenant, bkt := range b.buckets {
		if b.limits.QueryBudgetMode(tenant) == ModeDisabled {
			continue
		}
		if rate, burst := b.rateAndBurst(tenant); rate > 0 {
			levels[tenant] = bkt.levelAt(now, rate, burst)
		}
	}
	b.mtx.Unlock()

	for tenant, level := range levels {
		ch <- prometheus.MustNewConstMetric(levelDesc, prometheus.GaugeValue, level, tenant)
	}
}

// useBucketLocked returns the bucket of tenant refilled to now, and creates it full if it does not exist yet.
// The caller must hold the mutex of the Budget.
func (b *Budget) useBucketLocked(tenant string, now time.Time, rate, burst float64) *bucket {
	bkt, ok := b.buckets[tenant]
	if !ok {
		bkt = &bucket{level: burst, last: now}
		b.buckets[tenant] = bkt
	}
	bkt.refill(now, rate, burst)
	return bkt
}

// rateAndBurst returns the limits of the bucket of tenant. The burst is at least the rate, so a bucket always
// holds one second of refill.
func (b *Budget) rateAndBurst(tenant string) (rate, burst float64) {
	rate = float64(b.limits.QueryBudgetRate(tenant))
	burst = float64(b.limits.QueryBudgetBurst(tenant))
	return rate, max(burst, rate)
}

// cleanup removes idle tenants and their series.
func (b *Budget) cleanup(now time.Time) {
	removed := b.removeIdleBuckets(now)

	// deleting series scans all series of a metric, so it happens without holding the lock. a tenant that is
	// back already keeps its series
	for _, tenant := range removed {
		b.mtx.Lock()
		_, back := b.buckets[tenant]
		b.mtx.Unlock()
		if back {
			continue
		}

		labels := prometheus.Labels{labelTenant: tenant}
		b.rejected.DeletePartialMatch(labels)
		b.charged.DeletePartialMatch(labels)
		b.refunded.DeletePartialMatch(labels)
	}
}

// removeIdleBuckets removes the buckets that weren't used for idleTimeout, at most once per cleanupInterval, and
// returns their tenants. A bucket that is not full is kept, so that removing it doesn't reset its debt.
func (b *Budget) removeIdleBuckets(now time.Time) []string {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if now.Sub(b.lastCleanup) < cleanupInterval {
		return nil
	}
	b.lastCleanup = now

	var removed []string
	for tenant, bkt := range b.buckets {
		if now.Sub(bkt.last) < idleTimeout {
			continue
		}
		if rate, burst := b.rateAndBurst(tenant); rate > 0 && bkt.levelAt(now, rate, burst) < burst {
			continue
		}
		delete(b.buckets, tenant)
		removed = append(removed, tenant)
	}
	return removed
}

// Query charges the jobs of an admitted query to the buckets of its tenants. The debt of a bucket is capped at
// its burst, so a single huge query blocks its tenant for at most burst/rate seconds. The units a query is
// charged beyond the cap are forgiven, but later refunds and corrections of the query first take them back.
// This way the cap applies to what the query actually read, no matter in which order its jobs are charged and
// corrected. It is safe for concurrent use, and a nil Query charges nothing.
type Query struct {
	budget *Budget
	op     string
	// tenants are the tenants of the query that have a query budget. The map is not modified after Admit.
	tenants map[string]*queryTenant
}

// Charge takes the estimate of a job from the bucket of tenant when the job is sent to the queriers.
func (q *Query) Charge(tenant string, cost jobcost.Cost) {
	q.add(tenant, -float64(cost.Estimate))
}

// Refund gives the estimate of a job back to the bucket of tenant if the job was charged but failed before a
// querier took it, for example because the queue was full or the query was canceled.
func (q *Query) Refund(tenant string, cost jobcost.Cost) {
	q.add(tenant, float64(cost.Estimate))
}

// Settle corrects the charge of a completed job with the bytes it actually read. Only block jobs are settled,
// the estimate of live-store jobs is a fixed cost.
func (q *Query) Settle(tenant string, cost jobcost.Cost, actual uint64) {
	if cost.Kind != jobcost.KindBlock {
		return
	}
	q.add(tenant, float64(cost.Estimate)-float64(actual))
}

// add changes the bucket of tenant by delta, which is negative for charges. The change is counted even if the
// bucket is not changed, so that tenants in dry run and tenants without a rate report what they use.
func (q *Query) add(tenant string, delta float64) {
	if q == nil || delta == 0 {
		return
	}
	qt, ok := q.tenants[tenant]
	if !ok {
		return
	}

	b := q.budget
	if delta < 0 {
		b.charged.WithLabelValues(tenant, q.op).Add(-delta)
	} else {
		b.refunded.WithLabelValues(tenant, q.op).Add(delta)
	}

	rate, burst := b.rateAndBurst(tenant)
	if qt.dryRunRejected || rate == 0 {
		return
	}

	now := b.now()
	b.mtx.Lock()
	defer b.mtx.Unlock()

	bkt := b.useBucketLocked(tenant, now, rate, burst)
	if delta > 0 {
		// a refund first takes back what was forgiven, as the forgiven charge was never taken
		takenBack := min(delta, qt.forgiven)
		qt.forgiven -= takenBack
		bkt.level = min(bkt.level+delta-takenBack, burst)
		return
	}

	// the debt is capped at burst. if burst was lowered below the debt, the debt is kept, but a charge doesn't
	// add to it
	floor := min(bkt.level, -burst)
	level := bkt.level + delta
	qt.forgiven += max(floor-level, 0)
	bkt.level = max(level, floor)
}

// bucket is the token bucket of a tenant.
type bucket struct {
	// level is the number of units in the bucket. It is negative if the tenant used more than it had.
	level float64
	// last is when the bucket was last used and refilled.
	last time.Time
}

// refill adds the units the bucket gained since it was last used, and marks it as used now.
func (bkt *bucket) refill(now time.Time, rate, burst float64) {
	bkt.level = bkt.levelAt(now, rate, burst)
	bkt.last = now
}

// levelAt returns the level the bucket has at now. It never exceeds burst.
func (bkt *bucket) levelAt(now time.Time, rate, burst float64) float64 {
	level := bkt.level
	if elapsed := now.Sub(bkt.last); elapsed > 0 {
		level += rate * elapsed.Seconds()
	}
	return min(level, burst)
}

// retryAfter returns the time until a bucket with a negative level and rate is out of debt again.
func retryAfter(level, rate float64) time.Duration {
	d := time.Duration(-level / rate * float64(time.Second))
	return max(d, minRetryAfter)
}
