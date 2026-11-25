package querybudget

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/frontend/jobcost"
)

func TestAdmit(t *testing.T) {
	tests := []struct {
		name          string
		limits        map[string]testLimit
		chargedBefore map[string]uint64 // charged by an earlier query
		tenants       []string

		expected        Decision // without Query
		expectedTenants []string // the tenants of Decision.Query
		expectedDryRun  []string // the tenants of the query in dry run that would reject it
	}{
		{
			name:    "no limits",
			tenants: []string{"a"},
		},
		{
			name:          "disabled tenant with empty bucket",
			limits:        map[string]testLimit{"a": {mode: ModeDisabled, rate: 10, burst: 10}},
			chargedBefore: map[string]uint64{"a": 100},
			tenants:       []string{"a"},
		},
		{
			name:            "enabled tenant without rate is only counted",
			limits:          map[string]testLimit{"a": {mode: ModeEnabled}},
			chargedBefore:   map[string]uint64{"a": 100},
			tenants:         []string{"a"},
			expectedTenants: []string{"a"},
		},
		{
			name:            "enabled tenant with units left",
			limits:          map[string]testLimit{"a": {mode: ModeEnabled, rate: 10, burst: 100}},
			chargedBefore:   map[string]uint64{"a": 99},
			tenants:         []string{"a"},
			expectedTenants: []string{"a"},
		},
		{
			name:            "an empty bucket is not in debt",
			limits:          map[string]testLimit{"a": {mode: ModeEnabled, rate: 10, burst: 100}},
			chargedBefore:   map[string]uint64{"a": 100},
			tenants:         []string{"a"},
			expectedTenants: []string{"a"},
		},
		{
			name:          "enabled tenant in debt",
			limits:        map[string]testLimit{"a": {mode: ModeEnabled, rate: 10, burst: 100}},
			chargedBefore: map[string]uint64{"a": 150},
			tenants:       []string{"a"},
			expected:      Decision{Rejected: true, RetryAfter: 5 * time.Second},
		},
		{
			name:          "retry after is at least a second",
			limits:        map[string]testLimit{"a": {mode: ModeEnabled, rate: 1000, burst: 1000}},
			chargedBefore: map[string]uint64{"a": 1001},
			tenants:       []string{"a"},
			expected:      Decision{Rejected: true, RetryAfter: time.Second},
		},
		{
			name:            "dry run tenant in debt",
			limits:          map[string]testLimit{"a": {mode: ModeDryRun, rate: 10, burst: 100}},
			chargedBefore:   map[string]uint64{"a": 150},
			tenants:         []string{"a"},
			expected:        Decision{DryRunRejected: true},
			expectedTenants: []string{"a"},
			expectedDryRun:  []string{"a"},
		},
		{
			name: "all tenants of a query are checked",
			limits: map[string]testLimit{
				"a": {mode: ModeEnabled, rate: 10, burst: 100},
				"b": {mode: ModeEnabled, rate: 10, burst: 100},
			},
			chargedBefore: map[string]uint64{"b": 200},
			tenants:       []string{"a", "b"},
			expected:      Decision{Rejected: true, RetryAfter: 10 * time.Second},
		},
		{
			name: "retry after only depends on enabled tenants",
			limits: map[string]testLimit{
				"a": {mode: ModeDryRun, rate: 1, burst: 100},
				"b": {mode: ModeEnabled, rate: 10, burst: 100},
			},
			chargedBefore: map[string]uint64{"a": 200, "b": 150},
			tenants:       []string{"a", "b"},
			expected:      Decision{Rejected: true, RetryAfter: 5 * time.Second},
		},
		{
			name: "dry run is per tenant",
			limits: map[string]testLimit{
				"a": {mode: ModeDryRun, rate: 10, burst: 100},
				"b": {mode: ModeEnabled, rate: 10, burst: 100},
				"c": {mode: ModeDisabled},
			},
			chargedBefore:   map[string]uint64{"a": 200},
			tenants:         []string{"a", "b", "c"},
			expected:        Decision{DryRunRejected: true},
			expectedTenants: []string{"a", "b"},
			expectedDryRun:  []string{"a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := newTestBudget(tc.limits)
			for tenant, units := range tc.chargedBefore {
				q := b.Admit([]string{tenant}, jobcost.OpSearch).Query
				q.Charge(tenant, jobcost.Cost{Kind: jobcost.KindBlock, Estimate: units})
			}

			d := b.Admit(tc.tenants, jobcost.OpSearch)
			q := d.Query
			d.Query = nil
			require.Equal(t, tc.expected, d)

			if tc.expectedTenants == nil {
				require.Nil(t, q)
				return
			}
			require.NotNil(t, q)
			var tenants, dryRun []string
			for tenant, qt := range q.tenants {
				tenants = append(tenants, tenant)
				if qt.dryRunRejected {
					dryRun = append(dryRun, tenant)
				}
			}
			require.ElementsMatch(t, tc.expectedTenants, tenants)
			require.ElementsMatch(t, tc.expectedDryRun, dryRun)
		})
	}
}

func TestQueryChargeRefundSettle(t *testing.T) {
	block := func(estimate uint64) jobcost.Cost { return jobcost.Cost{Kind: jobcost.KindBlock, Estimate: estimate} }
	liveStore := jobcost.Cost{Kind: jobcost.KindLiveStore, Estimate: 100}
	times := func(n int, fn func(q *Query)) func(q *Query) {
		return func(q *Query) {
			for range n {
				fn(q)
			}
		}
	}

	type step struct {
		advance  time.Duration
		do       func(q *Query)
		expected float64 // level after the step
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "charge and refill up to burst",
			steps: []step{
				{do: func(q *Query) { q.Charge("a", block(300)) }, expected: 700},
				{advance: 10 * time.Second, expected: 800},
				{advance: time.Hour, expected: 1000},
			},
		},
		{
			name: "debt is capped at a full bucket",
			steps: []step{
				{do: func(q *Query) { q.Charge("a", block(5000)) }, expected: -1000},
				{advance: 50 * time.Second, expected: -500},
			},
		},
		{
			name: "refund is capped at burst",
			steps: []step{
				{do: func(q *Query) { q.Charge("a", block(100)) }, expected: 900},
				{do: func(q *Query) { q.Refund("a", block(500)) }, expected: 1000},
			},
		},
		{
			name: "block jobs are settled with the actual bytes",
			steps: []step{
				{do: func(q *Query) { q.Charge("a", block(100)) }, expected: 900},
				{do: func(q *Query) { q.Settle("a", block(100), 400) }, expected: 600},
				{do: func(q *Query) { q.Settle("a", block(100), 50) }, expected: 650},
			},
		},
		{
			name: "settles below the estimate first take back what the cap forgave",
			steps: []step{
				// 10000 charged, 8000 of them forgiven
				{do: times(100, func(q *Query) { q.Charge("a", block(100)) }), expected: -1000},
				// the jobs read 5000, which is still 3000 beyond the cap
				{do: times(100, func(q *Query) { q.Settle("a", block(100), 50) }), expected: -1000},
			},
		},
		{
			name: "refunds first take back what the cap forgave",
			steps: []step{
				{do: times(100, func(q *Query) { q.Charge("a", block(100)) }), expected: -1000},
				// half the jobs didn't run, the other half is still 3000 beyond the cap
				{do: times(50, func(q *Query) { q.Refund("a", block(100)) }), expected: -1000},
				// none of the jobs ran
				{do: times(50, func(q *Query) { q.Refund("a", block(100)) }), expected: 1000},
			},
		},
		{
			name: "settles above the estimate are capped too",
			steps: []step{
				{do: func(q *Query) { q.Charge("a", block(1500)) }, expected: -500},
				{do: func(q *Query) { q.Settle("a", block(1500), 3000) }, expected: -1000},
				{do: func(q *Query) { q.Refund("a", block(500)) }, expected: -1000},
				{do: func(q *Query) { q.Refund("a", block(1500)) }, expected: 0},
			},
		},
		{
			name: "live-store jobs keep their fixed cost",
			steps: []step{
				{do: func(q *Query) { q.Charge("a", liveStore) }, expected: 900},
				{do: func(q *Query) { q.Settle("a", liveStore, 1_000_000) }, expected: 900},
			},
		},
		{
			name: "jobs of tenants that are not part of the query are not charged",
			steps: []step{
				{do: func(q *Query) { q.Charge("b", block(300)) }, expected: 1000},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, clock := newTestBudget(map[string]testLimit{
				"a": {mode: ModeEnabled, rate: 10, burst: 1000},
				"b": {mode: ModeEnabled, rate: 10, burst: 1000},
			})
			q := b.Admit([]string{"a"}, jobcost.OpSearch).Query
			for i, s := range tc.steps {
				clock.advance(s.advance)
				if s.do != nil {
					s.do(q)
				}
				require.InDelta(t, s.expected, level(t, b, "a"), 0.001, "step %d", i)
			}
		})
	}
}

// the debt one query was forgiven doesn't change the charges of other queries
func TestQueriesForgiveSeparately(t *testing.T) {
	block := jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 1000}
	b, _ := newTestBudget(map[string]testLimit{"a": {mode: ModeEnabled, rate: 10, burst: 1000}})

	huge := b.Admit([]string{"a"}, jobcost.OpSearch).Query
	other := b.Admit([]string{"a"}, jobcost.OpSearch).Query
	other.Charge("a", block)
	for range 5 {
		huge.Charge("a", block)
	}
	require.InDelta(t, -1000, level(t, b, "a"), 0.001)

	// the job of the other query read nothing. its refund isn't taken by what huge was forgiven
	other.Settle("a", block, 0)
	require.InDelta(t, 0, level(t, b, "a"), 0.001)
}

func TestLoweringBurstKeepsDebt(t *testing.T) {
	block := func(estimate uint64) jobcost.Cost { return jobcost.Cost{Kind: jobcost.KindBlock, Estimate: estimate} }
	limits := testLimits{"a": {mode: ModeEnabled, rate: 1, burst: 100}}
	clock := &testClock{t: time.Unix(1000, 0)}
	b := New(limits, prometheus.NewRegistry())
	b.now = clock.now

	q := b.Admit([]string{"a"}, jobcost.OpSearch).Query
	q.Charge("a", block(250))
	require.InDelta(t, -100, level(t, b, "a"), 0.001)

	// the debt above the new burst is kept, but charges don't add to it
	limits["a"] = testLimit{mode: ModeEnabled, rate: 1, burst: 50}
	q.Charge("a", block(10))
	require.InDelta(t, -100, level(t, b, "a"), 0.001)

	// refunds first take back the 60 units that were forgiven: 100 - 260 + 200
	q.Refund("a", block(200))
	require.InDelta(t, 40, level(t, b, "a"), 0.001)
	// all charges are refunded, the bucket is full with the new burst
	q.Refund("a", block(60))
	require.InDelta(t, 50, level(t, b, "a"), 0.001)
}

// a dry run rejection isn't counted for a query that an enabled tenant rejected
func TestDryRunRejectionsAreNotCountedForRejectedQueries(t *testing.T) {
	block := jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 200}
	b, _ := newTestBudget(map[string]testLimit{
		"dry":     {mode: ModeDryRun, rate: 10, burst: 100},
		"enabled": {mode: ModeEnabled, rate: 10, burst: 100},
	})
	q := b.Admit([]string{"dry", "enabled"}, jobcost.OpSearch).Query
	q.Charge("dry", block)
	q.Charge("enabled", block)

	require.True(t, b.Admit([]string{"dry", "enabled"}, jobcost.OpSearch).Rejected)
	require.True(t, b.Admit([]string{"dry"}, jobcost.OpSearch).DryRunRejected)

	require.InDelta(t, 1.0, testutil.ToFloat64(b.rejected.WithLabelValues("dry", "search", ModeDryRun)), 0.001)
	require.InDelta(t, 1.0, testutil.ToFloat64(b.rejected.WithLabelValues("enabled", "search", ModeEnabled)), 0.001)
}

// a running query keeps the bucket of its tenant
func TestCleanupKeepsBucketsInUse(t *testing.T) {
	block := jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 100}
	b, clock := newTestBudget(map[string]testLimit{
		"a":     {mode: ModeEnabled, rate: 1000, burst: 1000},
		"other": {mode: ModeEnabled, rate: 1000, burst: 1000},
	})
	q := b.Admit([]string{"a"}, jobcost.OpSearch).Query

	clock.advance(idleTimeout - time.Second)
	q.Charge("a", block)
	q.Settle("a", block, 0) // the bucket is full again
	clock.advance(time.Second)
	b.Admit([]string{"other"}, jobcost.OpSearch)

	require.ElementsMatch(t, []string{"a", "other"}, tenantsOf(b))
}

func TestDryRunRejectedTenantsDontChangeTheirBucket(t *testing.T) {
	block := func(estimate uint64) jobcost.Cost { return jobcost.Cost{Kind: jobcost.KindBlock, Estimate: estimate} }
	b, _ := newTestBudget(map[string]testLimit{
		"dry":     {mode: ModeDryRun, rate: 10, burst: 1000},
		"enabled": {mode: ModeEnabled, rate: 10, burst: 1000},
	})
	b.Admit([]string{"dry"}, jobcost.OpSearch).Query.Charge("dry", block(1500))
	require.InDelta(t, -500, level(t, b, "dry"), 0.001)

	q := b.Admit([]string{"dry", "enabled"}, jobcost.OpSearch).Query
	for _, tenant := range []string{"dry", "enabled"} {
		q.Charge(tenant, block(300))
		q.Settle(tenant, block(300), 600)
	}

	require.InDelta(t, -500, level(t, b, "dry"), 0.001)
	require.InDelta(t, 400, level(t, b, "enabled"), 0.001)
	require.InDelta(t, 2100.0, testutil.ToFloat64(b.charged.WithLabelValues("dry", "search")), 0.001)
}

func TestChargedAndRefundedCounters(t *testing.T) {
	block := jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 100}
	b, _ := newTestBudget(map[string]testLimit{
		"enabled":  {mode: ModeEnabled, rate: 10, burst: 100},
		"dry-run":  {mode: ModeDryRun},
		"disabled": {mode: ModeDisabled, rate: 10, burst: 100},
	})

	q := b.Admit([]string{"enabled", "dry-run", "disabled"}, jobcost.OpSearch).Query
	for _, tenant := range []string{"enabled", "dry-run", "disabled"} {
		q.Charge(tenant, block)
		q.Settle(tenant, block, 250)
		q.Refund(tenant, block)
	}

	// a tenant in dry run without a rate still reports what it uses, a disabled tenant reports nothing
	for _, tenant := range []string{"enabled", "dry-run"} {
		require.InDelta(t, 250.0, testutil.ToFloat64(b.charged.WithLabelValues(tenant, "search")), 0.001, tenant)
		require.InDelta(t, 100.0, testutil.ToFloat64(b.refunded.WithLabelValues(tenant, "search")), 0.001, tenant)
	}
	require.Zero(t, seriesOf(t, b.charged, "disabled"))
}

func TestRejectedCounter(t *testing.T) {
	b, _ := newTestBudget(map[string]testLimit{
		"a": {mode: ModeEnabled, rate: 10, burst: 100},
		"b": {mode: ModeDryRun, rate: 10, burst: 100},
	})
	q := b.Admit([]string{"a", "b"}, jobcost.OpTraceByID).Query
	q.Charge("a", jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 200})
	q.Charge("b", jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 200})

	b.Admit([]string{"a"}, jobcost.OpTraceByID)
	b.Admit([]string{"b"}, jobcost.OpTraceByID)
	b.Admit([]string{"b"}, jobcost.OpTraceByID)

	require.InDelta(t, 1.0, testutil.ToFloat64(b.rejected.WithLabelValues("a", "traces", ModeEnabled)), 0.001)
	require.InDelta(t, 2.0, testutil.ToFloat64(b.rejected.WithLabelValues("b", "traces", ModeDryRun)), 0.001)
}

func TestCleanupRemovesIdleFullBuckets(t *testing.T) {
	b, clock := newTestBudget(map[string]testLimit{
		"full":    {mode: ModeEnabled, rate: 1, burst: 1000},
		"debt":    {mode: ModeEnabled, rate: 1, burst: 1000},
		"no-rate": {mode: ModeDryRun},
		"active":  {mode: ModeEnabled, rate: 1, burst: 1000},
	})
	q := b.Admit([]string{"full", "debt", "no-rate"}, jobcost.OpSearch).Query
	q.Charge("debt", jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 2000})
	q.Charge("no-rate", jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 10})

	clock.advance(idleTimeout)
	b.Admit([]string{"active"}, jobcost.OpSearch)

	require.ElementsMatch(t, []string{"debt", "active"}, tenantsOf(b))
	require.Zero(t, seriesOf(t, b.charged, "no-rate"))

	// the debt is kept, not forgiven: 1000 units of debt left minus 900 seconds of refill
	require.InDelta(t, -100.0, level(t, b, "debt"), 0.001)
}

func TestLevelMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	limits := testLimits{
		"a":        {mode: ModeEnabled, rate: 10, burst: 100},
		"no-rate":  {mode: ModeDryRun},
		"disabled": {mode: ModeEnabled, rate: 10, burst: 100},
	}
	clock := &testClock{t: time.Unix(1000, 0)}
	b := New(limits, reg)
	b.now = clock.now

	b.Admit([]string{"a", "no-rate", "disabled"}, jobcost.OpSearch).Query.Charge("a", jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 150})
	clock.advance(2 * time.Second)
	// a tenant that is disabled after it was admitted isn't reported anymore
	limits["disabled"] = testLimit{mode: ModeDisabled, rate: 10, burst: 100}

	expected := `
# HELP tempo_query_frontend_query_budget_level Units left in the query budget bucket of a tenant. Negative values are debt that has to refill before queries are admitted again.
# TYPE tempo_query_frontend_query_budget_level gauge
tempo_query_frontend_query_budget_level{tenant="a"} -30
`
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(expected), "tempo_query_frontend_query_budget_level"))
}

func TestNilQueryAndLimits(t *testing.T) {
	var q *Query
	q.Charge("a", jobcost.Cost{Estimate: 1})
	q.Refund("a", jobcost.Cost{Estimate: 1})
	q.Settle("a", jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 1}, 2)

	b := New(nil, nil)
	require.Equal(t, Decision{}, b.Admit([]string{"a"}, jobcost.OpSearch))
	b.Collect(make(chan prometheus.Metric, 1))
}

// BenchmarkChargeAndSettle charges and settles jobs of many tenants concurrently, which all share the mutex of
// the Budget.
func BenchmarkChargeAndSettle(b *testing.B) {
	limits := testLimits{}
	tenants := make([]string, 100)
	for i := range tenants {
		tenants[i] = fmt.Sprintf("tenant-%d", i)
		limits[tenants[i]] = testLimit{mode: ModeEnabled, rate: 1 << 30, burst: 1 << 40}
	}
	budget := New(limits, prometheus.NewRegistry())
	queries := make([]*Query, len(tenants))
	for i, tenant := range tenants {
		queries[i] = budget.Admit([]string{tenant}, jobcost.OpSearch).Query
	}
	cost := jobcost.Cost{Kind: jobcost.KindBlock, Estimate: 1 << 20}

	var next atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		i := int(next.Add(1))
		for pb.Next() {
			i = (i + 1) % len(tenants)
			queries[i].Charge(tenants[i], cost)
			queries[i].Settle(tenants[i], cost, 1<<19)
		}
	})
}

type testLimit struct {
	mode        string
	rate, burst uint64
}

type testLimits map[string]testLimit

func (l testLimits) QueryBudgetMode(tenant string) string  { return l[tenant].mode }
func (l testLimits) QueryBudgetRate(tenant string) uint64  { return l[tenant].rate }
func (l testLimits) QueryBudgetBurst(tenant string) uint64 { return l[tenant].burst }

type testClock struct {
	t time.Time
}

func (c *testClock) now() time.Time          { return c.t }
func (c *testClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestBudget(limits map[string]testLimit) (*Budget, *testClock) {
	clock := &testClock{t: time.Unix(1000, 0)}
	var l Limits
	if limits != nil {
		l = testLimits(limits)
	}
	b := New(l, prometheus.NewRegistry())
	b.now = clock.now
	return b, clock
}

// level returns the level of the bucket of tenant at now.
func level(t *testing.T, b *Budget, tenant string) float64 {
	t.Helper()

	b.mtx.Lock()
	defer b.mtx.Unlock()
	bkt, ok := b.buckets[tenant]
	require.True(t, ok, "tenant %s has no bucket", tenant)

	rate, burst := b.rateAndBurst(tenant)
	return bkt.levelAt(b.now(), rate, burst)
}

// seriesOf returns the number of series of c with the label tenant.
func seriesOf(t *testing.T, c prometheus.Collector, tenant string) int {
	t.Helper()

	reg := prometheus.NewPedanticRegistry()
	require.NoError(t, reg.Register(c))
	families, err := reg.Gather()
	require.NoError(t, err)

	n := 0
	for _, f := range families {
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == labelTenant && l.GetValue() == tenant {
					n++
				}
			}
		}
	}
	return n
}

func tenantsOf(b *Budget) []string {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	tenants := make([]string, 0, len(b.buckets))
	for tenant := range b.buckets {
		tenants = append(tenants, tenant)
	}
	return tenants
}
