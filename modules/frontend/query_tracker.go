package frontend

import (
	"context"
	_ "embed"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/tenant"
	"github.com/grafana/tempo/v3/pkg/util"
	tempo_log "github.com/grafana/tempo/v3/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type trackedQuery struct {
	ID        uint64    `json:"id"`
	Tenant    string    `json:"tenant"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Query     string    `json:"query,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

type queryTrackerTenant struct {
	Name    string `json:"name"`
	Active  int    `json:"active"`
	Dropped uint64 `json:"dropped"`
}

type tracker interface {
	track(context.Context, string, string, func() string) func()
}

type noOpQueryTracker struct{}

func (noOpQueryTracker) track(context.Context, string, string, func() string) func() {
	return func() {}
}

type queryTracker struct {
	nextID              atomic.Uint64
	tenants             sync.Map
	maxQueriesPerTenant int
	maxQueryTextBytes   int
	droppedQueries      *prometheus.CounterVec
	droppedLogger       *tempo_log.RateLimitedLogger
}

type tenantQueryRegistry struct {
	queries sync.Map
	active  atomic.Int64
	dropped atomic.Uint64
}

func newQueryTracker(cfg QueryTrackerConfig, logger log.Logger, registerer prometheus.Registerer) tracker {
	if !cfg.Enabled {
		return noOpQueryTracker{}
	}
	return &queryTracker{
		maxQueriesPerTenant: cfg.MaxQueriesPerTenant,
		maxQueryTextBytes:   cfg.MaxQueryTextBytes,
		droppedQueries: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
			Name: "tempo_query_frontend_query_tracker_dropped_total",
			Help: "Total number of query tracking entries dropped because the per-tenant limit was reached. The queries still execute.",
		}, []string{metricLabelTenant}),
		droppedLogger: tempo_log.NewRateLimitedLogger(1, level.Warn(logger)),
	}
}

func (t *queryTracker) track(ctx context.Context, method, path string, queryText func() string) func() {
	tenantIDs, err := tenant.TenantIDs(ctx)
	if err != nil {
		return func() {}
	}
	query := truncateQuery(queryText(), t.maxQueryTextBytes)

	type trackedQueryEntry struct {
		registry *tenantQueryRegistry
		id       uint64
	}

	entries := make([]trackedQueryEntry, 0, len(tenantIDs))
	for _, tenantID := range tenantIDs {
		value, _ := t.tenants.LoadOrStore(tenantID, &tenantQueryRegistry{})
		registry := value.(*tenantQueryRegistry)
		if !registry.reserve(t.maxQueriesPerTenant) {
			registry.dropped.Add(1)
			t.droppedQueries.WithLabelValues(tenantID).Inc()
			t.droppedLogger.Log("msg", "query tracker capacity reached; query continues without tracking", "tenant", tenantID, "max_queries_per_tenant", t.maxQueriesPerTenant)
			continue
		}

		id := t.nextID.Add(1)
		registry.queries.Store(id, trackedQuery{
			ID:        id,
			Tenant:    tenantID,
			Method:    method,
			Path:      path,
			Query:     query,
			StartedAt: time.Now().UTC(),
		})
		entries = append(entries, trackedQueryEntry{registry: registry, id: id})
	}

	return func() {
		for _, entry := range entries {
			if _, loaded := entry.registry.queries.LoadAndDelete(entry.id); loaded {
				entry.registry.active.Add(-1)
			}
		}
	}
}

func (r *tenantQueryRegistry) reserve(maxQueries int) bool {
	for {
		active := r.active.Load()
		if maxQueries > 0 && active >= int64(maxQueries) {
			return false
		}
		if r.active.CompareAndSwap(active, active+1) {
			return true
		}
	}
}

func truncateQuery(query string, maxBytes int) string {
	if maxBytes <= 0 || len(query) <= maxBytes {
		return query
	}
	return strings.ToValidUTF8(query[:maxBytes], "")
}

func (t *queryTracker) trackedQueries(tenant string) []trackedQuery {
	value, ok := t.tenants.Load(tenant)
	if !ok {
		return nil
	}
	registry := value.(*tenantQueryRegistry)
	queries := make([]trackedQuery, 0)
	registry.queries.Range(func(_, value any) bool {
		queries = append(queries, value.(trackedQuery))
		return true
	})

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].ID < queries[j].ID
	})
	return queries
}

func (t *queryTracker) trackedTenants() []queryTrackerTenant {
	result := make([]queryTrackerTenant, 0)
	t.tenants.Range(func(key, value any) bool {
		registry := value.(*tenantQueryRegistry)
		active := registry.active.Load()
		dropped := registry.dropped.Load()
		if active > 0 {
			result = append(result, queryTrackerTenant{
				Name:    key.(string),
				Active:  int(active),
				Dropped: dropped,
			})
		}
		return true
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

//go:embed query_tracker.gohtml
var queryTrackerPageHTML string

var queryTrackerTemplate = template.Must(template.New("query-tracker").Parse(queryTrackerPageHTML))

type queryTrackerPage struct {
	Now                 time.Time            `json:"now"`
	Tenant              string               `json:"tenant,omitempty"`
	Tenants             []queryTrackerTenant `json:"tenants,omitempty"`
	Queries             []trackedQuery       `json:"queries,omitempty"`
	MaxQueriesPerTenant int                  `json:"max_queries_per_tenant"`
	MaxQueryTextBytes   int                  `json:"max_query_text_bytes"`
}

func (t *queryTracker) tenantsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.RenderHTTPResponse(w, queryTrackerPage{Now: time.Now().UTC(), Tenants: t.trackedTenants(), MaxQueriesPerTenant: t.maxQueriesPerTenant, MaxQueryTextBytes: t.maxQueryTextBytes}, queryTrackerTemplate, r)
	})
}

func (t *queryTracker) tenantHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := mux.Vars(r)["tenant"]
		if tenantID == "" {
			http.Error(w, "tenant ID can't be empty", http.StatusBadRequest)
			return
		}
		util.RenderHTTPResponse(w, queryTrackerPage{Now: time.Now().UTC(), Tenant: tenantID, Queries: t.trackedQueries(tenantID), MaxQueriesPerTenant: t.maxQueriesPerTenant, MaxQueryTextBytes: t.maxQueryTextBytes}, queryTrackerTemplate, r)
	})
}
