package querier

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"google.golang.org/grpc/health/grpc_health_v1"

	cortex_client "github.com/cortexproject/cortex/pkg/ingester/client"
	"github.com/cortexproject/cortex/pkg/ring"
	"github.com/cortexproject/cortex/pkg/util"

	"github.com/grafana/frigg/pkg/friggpb"
	"github.com/grafana/frigg/pkg/ingester/client"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/grafana/frigg/pkg/util/validation"
)

var readinessProbeSuccess = []byte("Ready")

// Querier handlers queries.
type Querier struct {
	cfg    Config
	ring   ring.ReadRing
	pool   *cortex_client.Pool
	store  storage.Store
	limits *validation.Overrides
}

type responseFromIngesters struct {
	addr     string
	response interface{}
}

// New makes a new Querier.
func New(cfg Config, clientCfg client.Config, ring ring.ReadRing, store storage.Store, limits *validation.Overrides) (*Querier, error) {
	factory := func(addr string) (grpc_health_v1.HealthClient, error) {
		return client.New(clientCfg, addr)
	}

	return newQuerier(cfg, clientCfg, factory, ring, store, limits)
}

// newQuerier creates a new Querier and allows to pass a custom ingester client factory
// used for testing purposes
func newQuerier(cfg Config, clientCfg client.Config, clientFactory cortex_client.Factory, ring ring.ReadRing, store storage.Store, limits *validation.Overrides) (*Querier, error) {
	return &Querier{
		cfg:    cfg,
		ring:   ring,
		pool:   cortex_client.NewPool(clientCfg.PoolConfig, ring, clientFactory, util.Logger),
		store:  store,
		limits: limits,
	}, nil
}

// FindTraceByID implements friggpb.Querier.
func (i *Querier) FindTraceByID(ctx context.Context, req *friggpb.TraceByIDRequest) (*friggpb.TraceByIDResponse, error) {
	if !validation.ValidTraceID(req.TraceID) {
		return nil, fmt.Errorf("invalid trace id")
	}

	// jpe - something here

	return &friggpb.TraceByIDResponse{
		Trace: nil,
	}, nil
}

// forGivenIngesters runs f, in parallel, for given ingesters
// TODO taken from Cortex, see if we can refactor out an usable interface.
func (q *Querier) forGivenIngesters(ctx context.Context, replicationSet ring.ReplicationSet, f func(friggpb.QuerierClient) (interface{}, error)) ([]responseFromIngesters, error) {
	results, err := replicationSet.Do(ctx, q.cfg.ExtraQueryDelay, func(ingester *ring.IngesterDesc) (interface{}, error) {
		client, err := q.pool.GetClientFor(ingester.Addr)
		if err != nil {
			return nil, err
		}

		resp, err := f(client.(friggpb.QuerierClient))
		if err != nil {
			return nil, err
		}

		return responseFromIngesters{ingester.Addr, resp}, nil
	})
	if err != nil {
		return nil, err
	}

	responses := make([]responseFromIngesters, 0, len(results))
	for _, result := range results {
		responses = append(responses, result.(responseFromIngesters))
	}

	return responses, err
}

// ReadinessHandler is used to indicate to k8s when the querier is ready.
// Returns 200 when the querier is ready, 500 otherwise.
func (q *Querier) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	_, err := q.ring.GetAll()
	if err != nil {
		http.Error(w, "Not ready: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(readinessProbeSuccess); err != nil {
		level.Error(util.Logger).Log("msg", "error writing success message", "error", err)
	}
}

// Check implements the grpc healthcheck
func (*Querier) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
