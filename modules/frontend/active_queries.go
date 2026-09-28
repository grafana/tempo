package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
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

type activeQueryTracker struct {
	nextID  atomic.Uint64
	mu      sync.RWMutex
	queries map[uint64]activeQuery
	logger  log.Logger
}

func newActiveQueryTracker(logger log.Logger) *activeQueryTracker {
	return &activeQueryTracker{
		queries: make(map[uint64]activeQuery),
		logger:  logger,
	}
}

func (t *activeQueryTracker) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer t.track(r.Context(), r.Method, r.URL.Path, r.URL.RawQuery)()
		next.ServeHTTP(w, r)
	})
}

func (t *activeQueryTracker) track(ctx context.Context, method, path, query string) func() {
	id := t.nextID.Add(1)
	tenant, _ := user.ExtractOrgID(ctx)
	t.mu.Lock()
	t.queries[id] = activeQuery{
		ID:        id,
		Tenant:    tenant,
		Method:    method,
		Path:      path,
		Query:     query,
		StartedAt: time.Now().UTC(),
	}
	t.mu.Unlock()

	return func() {
		t.mu.Lock()
		delete(t.queries, id)
		t.mu.Unlock()
	}
}

func (t *activeQueryTracker) activeQueries(tenant string) []activeQuery {
	t.mu.RLock()
	queries := make([]activeQuery, 0, len(t.queries))
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

func (t *activeQueryTracker) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, err := user.ExtractOrgID(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Queries []activeQuery `json:"queries"`
		}{Queries: t.activeQueries(tenant)}); err != nil {
			level.Error(t.logger).Log("msg", "failed to encode active queries response", "err", err)
		}
	})
}
