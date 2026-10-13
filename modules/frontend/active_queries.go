package frontend

import (
	"context"
	_ "embed"
	"html/template"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/pkg/util"
)

// activeQuery is the public representation of a query currently being processed
// by this query-frontend instance.
type activeQuery struct {
	ID        uint64    `json:"id"`
	Tenant    string    `json:"tenant"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Query     string    `json:"query,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

type activeQueryTenant struct {
	Name    string `json:"name"`
	Active  int    `json:"active"`
	Dropped uint64 `json:"dropped"`
}

type queryTracker interface {
	track(context.Context, string, string, string) func()
}

type noopQueryTracker struct{}

func (noopQueryTracker) track(context.Context, string, string, string) func() { return func() {} }

type activeQueryTracker struct {
	nextID              atomic.Uint64
	mu                  sync.RWMutex
	queries             map[uint64]activeQuery
	dropped             map[string]uint64
	maxQueriesPerTenant int
	maxQueryTextBytes   int
}

func newActiveQueryTracker(cfg QueryTrackerConfig) *activeQueryTracker {
	return &activeQueryTracker{
		queries:             make(map[uint64]activeQuery),
		dropped:             make(map[string]uint64),
		maxQueriesPerTenant: cfg.MaxQueriesPerTenant,
		maxQueryTextBytes:   cfg.MaxQueryTextBytes,
	}
}

func newQueryTracker(cfg QueryTrackerConfig) queryTracker {
	if !cfg.Enabled {
		return noopQueryTracker{}
	}
	return newActiveQueryTracker(cfg)
}

func (t *activeQueryTracker) track(ctx context.Context, method, path, query string) func() {
	tenant, err := user.ExtractOrgID(ctx)
	if err != nil {
		return func() {}
	}
	t.mu.Lock()
	active := 0
	for _, tracked := range t.queries {
		if tracked.Tenant == tenant {
			active++
		}
	}
	if t.maxQueriesPerTenant > 0 && active >= t.maxQueriesPerTenant {
		t.dropped[tenant]++
		t.mu.Unlock()
		return func() {}
	}

	id := t.nextID.Add(1)
	t.queries[id] = activeQuery{
		ID:        id,
		Tenant:    tenant,
		Method:    method,
		Path:      path,
		Query:     truncateQuery(query, t.maxQueryTextBytes),
		StartedAt: time.Now().UTC(),
	}
	t.mu.Unlock()

	return func() {
		t.mu.Lock()
		delete(t.queries, id)
		t.mu.Unlock()
	}
}

func truncateQuery(query string, maxBytes int) string {
	if maxBytes <= 0 || len(query) <= maxBytes {
		return query
	}
	query = query[:maxBytes]
	for !utf8.ValidString(query) {
		query = query[:len(query)-1]
	}
	return query
}

func (t *activeQueryTracker) activeQueries(tenant string) []activeQuery {
	t.mu.RLock()
	queries := make([]activeQuery, 0)
	for _, query := range t.queries {
		if query.Tenant == tenant {
			queries = append(queries, query)
		}
	}
	t.mu.RUnlock()

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].ID < queries[j].ID
	})
	return queries
}

func (t *activeQueryTracker) tenants() []activeQueryTenant {
	t.mu.RLock()
	tenants := make(map[string]activeQueryTenant)
	for _, query := range t.queries {
		tenant := tenants[query.Tenant]
		tenant.Name = query.Tenant
		tenant.Active++
		tenants[query.Tenant] = tenant
	}
	for name, dropped := range t.dropped {
		tenant := tenants[name]
		tenant.Name = name
		tenant.Dropped = dropped
		tenants[name] = tenant
	}
	t.mu.RUnlock()

	result := make([]activeQueryTenant, 0, len(tenants))
	for _, tenant := range tenants {
		result = append(result, tenant)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

//go:embed active_queries.gohtml
var activeQueriesPageHTML string

var activeQueriesTemplate = template.Must(template.New("active-queries").Parse(activeQueriesPageHTML))

type activeQueriesPage struct {
	Now                 time.Time           `json:"now"`
	Tenant              string              `json:"tenant,omitempty"`
	Tenants             []activeQueryTenant `json:"tenants,omitempty"`
	Queries             []activeQuery       `json:"queries,omitempty"`
	MaxQueriesPerTenant int                 `json:"max_queries_per_tenant"`
	MaxQueryTextBytes   int                 `json:"max_query_text_bytes"`
}

func (t *activeQueryTracker) tenantsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.RenderHTTPResponse(w, activeQueriesPage{Now: time.Now(), Tenants: t.tenants(), MaxQueriesPerTenant: t.maxQueriesPerTenant, MaxQueryTextBytes: t.maxQueryTextBytes}, activeQueriesTemplate, r)
	})
}

func (t *activeQueryTracker) tenantHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant := mux.Vars(r)["tenant"]
		if tenant == "" {
			http.Error(w, "tenant ID can't be empty", http.StatusBadRequest)
			return
		}
		util.RenderHTTPResponse(w, activeQueriesPage{Now: time.Now(), Tenant: tenant, Queries: t.activeQueries(tenant), MaxQueriesPerTenant: t.maxQueriesPerTenant, MaxQueryTextBytes: t.maxQueryTextBytes}, activeQueriesTemplate, r)
	})
}
