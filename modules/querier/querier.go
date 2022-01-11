package querier

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	cortex_worker "github.com/cortexproject/cortex/pkg/querier/worker"
	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/grafana/dskit/ring"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grafana/dskit/services"
	ingester_client "github.com/grafana/tempo/modules/ingester/client"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/search"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	httpgrpc_server "github.com/weaveworks/common/httpgrpc/server"
	"github.com/weaveworks/common/user"
	"go.uber.org/multierr"
)

var (
	metricIngesterClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "querier_ingester_clients",
		Help:      "The current number of ingester clients.",
	})
	metricEndpointDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "querier_external_endpoint_duration_seconds",
		Help:      "The duration of the external endpoints.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"endpoint"})
)

// Querier handlers queries.
type Querier struct {
	services.Service

	cfg    Config
	ring   ring.ReadRing
	pool   *ring_client.Pool
	store  storage.Store
	limits *overrides.Overrides

	subservices        *services.Manager
	subservicesWatcher *services.FailureWatcher
}

type responseFromIngesters struct {
	addr     string
	response interface{}
}

// New makes a new Querier.
func New(cfg Config, clientCfg ingester_client.Config, ring ring.ReadRing, store storage.Store, limits *overrides.Overrides) (*Querier, error) {
	factory := func(addr string) (ring_client.PoolClient, error) {
		return ingester_client.New(addr, clientCfg)
	}

	q := &Querier{
		cfg:  cfg,
		ring: ring,
		pool: ring_client.NewPool("querier_pool",
			clientCfg.PoolConfig,
			ring_client.NewRingServiceDiscovery(ring),
			factory,
			metricIngesterClients,
			log.Logger),
		store:  store,
		limits: limits,
	}

	q.Service = services.NewBasicService(q.starting, q.running, q.stopping)
	return q, nil
}

func (q *Querier) CreateAndRegisterWorker(handler http.Handler) error {
	q.cfg.Worker.MaxConcurrentRequests = q.cfg.MaxConcurrentQueries
	worker, err := cortex_worker.NewQuerierWorker(
		q.cfg.Worker,
		httpgrpc_server.NewServer(handler),
		log.Logger,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create frontend worker: %w", err)
	}

	return q.RegisterSubservices(worker, q.pool)
}

func (q *Querier) RegisterSubservices(s ...services.Service) error {
	var err error
	q.subservices, err = services.NewManager(s...)
	q.subservicesWatcher = services.NewFailureWatcher()
	q.subservicesWatcher.WatchManager(q.subservices)
	return err
}

func (q *Querier) starting(ctx context.Context) error {
	if q.subservices != nil {
		err := services.StartManagerAndAwaitHealthy(ctx, q.subservices)
		if err != nil {
			return fmt.Errorf("failed to start subservices %w", err)
		}
	}

	return nil
}

func (q *Querier) running(ctx context.Context) error {
	if q.subservices != nil {
		select {
		case <-ctx.Done():
			return nil
		case err := <-q.subservicesWatcher.Chan():
			return fmt.Errorf("subservices failed %w", err)
		}
	} else {
		<-ctx.Done()
	}
	return nil
}

func (q *Querier) stopping(_ error) error {
	if q.subservices != nil {
		return services.StopManagerAndAwaitStopped(context.Background(), q.subservices)
	}
	return nil
}

// FindTraceByID implements tempopb.Querier.
func (q *Querier) FindTraceByID(ctx context.Context, req *tempopb.TraceByIDRequest) (*tempopb.TraceByIDResponse, error) {
	if !validation.ValidTraceID(req.TraceID) {
		return nil, fmt.Errorf("invalid trace id")
	}

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.FindTraceByID")
	}

	span, ctx := opentracing.StartSpanFromContext(ctx, "Querier.FindTraceByID")
	defer span.Finish()

	var completeTrace *tempopb.Trace
	var spanCount, spanCountTotal, traceCountTotal int
	if req.QueryMode == QueryModeIngesters || req.QueryMode == QueryModeAll {
		replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
		if err != nil {
			return nil, errors.Wrap(err, "error finding ingesters in Querier.FindTraceByID")
		}

		span.LogFields(ot_log.String("msg", "searching ingesters"))
		// get responses from all ingesters in parallel
		responses, err := q.forGivenIngesters(ctx, replicationSet, func(client tempopb.QuerierClient) (interface{}, error) {
			return client.FindTraceByID(opentracing.ContextWithSpan(ctx, span), req)
		})
		if err != nil {
			return nil, errors.Wrap(err, "error querying ingesters in Querier.FindTraceByID")
		}

		for _, r := range responses {
			t := r.response.(*tempopb.TraceByIDResponse).Trace
			if t != nil {
				completeTrace, spanCount = trace.CombineTraceProtos(completeTrace, t)
				spanCountTotal += spanCount
				traceCountTotal++
			}
		}
		span.LogFields(ot_log.String("msg", "done searching ingesters"),
			ot_log.Bool("found", completeTrace != nil),
			ot_log.Int("combinedSpans", spanCountTotal),
			ot_log.Int("combinedTraces", traceCountTotal))
	}

	var failedBlocks int
	if req.QueryMode == QueryModeBlocks || req.QueryMode == QueryModeAll {
		span.LogFields(ot_log.String("msg", "searching store"))
		partialTraces, dataEncodings, blockErrs, err := q.store.Find(opentracing.ContextWithSpan(ctx, span), userID, req.TraceID, req.BlockStart, req.BlockEnd)
		if err != nil {
			return nil, errors.Wrap(err, "error querying store in Querier.FindTraceByID")
		}

		if blockErrs != nil {
			failedBlocks = len(blockErrs)
			_ = level.Warn(log.Logger).Log("msg", fmt.Sprintf("failed to query %d blocks", failedBlocks), "blockErrs", multierr.Combine(blockErrs...))
		}

		span.LogFields(ot_log.String("msg", "done searching store"))

		if len(partialTraces) != 0 {
			traceCountTotal = 0
			spanCountTotal = 0
			var storeTrace *tempopb.Trace

			for i, partialTrace := range partialTraces {
				storeTrace, err = model.CombineForRead(partialTrace, dataEncodings[i], storeTrace)
				if err != nil {
					return nil, errors.Wrap(err, "error combining in Querier.FindTraceByID")
				}
			}

			completeTrace, spanCount = trace.CombineTraceProtos(completeTrace, storeTrace)
			spanCountTotal += spanCount
			traceCountTotal++

			span.LogFields(ot_log.String("msg", "combined trace protos from store"),
				ot_log.Bool("found", completeTrace != nil),
				ot_log.Int("combinedSpans", spanCountTotal),
				ot_log.Int("combinedTraces", traceCountTotal))
		}
	}

	return &tempopb.TraceByIDResponse{
		Trace: completeTrace,
		Metrics: &tempopb.TraceByIDMetrics{
			FailedBlocks: uint32(failedBlocks),
		},
	}, nil
}

// forGivenIngesters runs f, in parallel, for given ingesters
func (q *Querier) forGivenIngesters(ctx context.Context, replicationSet ring.ReplicationSet, f func(client tempopb.QuerierClient) (interface{}, error)) ([]responseFromIngesters, error) {
	results, err := replicationSet.Do(ctx, q.cfg.ExtraQueryDelay, func(ctx context.Context, ingester *ring.InstanceDesc) (interface{}, error) {
		client, err := q.pool.GetClientFor(ingester.Addr)
		if err != nil {
			return nil, err
		}

		resp, err := f(client.(tempopb.QuerierClient))
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

func (q *Querier) SearchRecent(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	_, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.Search")
	}

	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.Search")
	}

	responses, err := q.forGivenIngesters(ctx, replicationSet, func(client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchRecent(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.Search")
	}

	return q.postProcessSearchResults(req, responses), nil
}

func (q *Querier) SearchTags(ctx context.Context, req *tempopb.SearchTagsRequest) (*tempopb.SearchTagsResponse, error) {
	_, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.SearchTags")
	}

	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.SearchTags")
	}

	// Get results from all ingesters
	lookupResults, err := q.forGivenIngesters(ctx, replicationSet, func(client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTags(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.SearchTags")
	}

	// Collect only unique values
	uniqueMap := map[string]struct{}{}
	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagsResponse).TagNames {
			uniqueMap[res] = struct{}{}
		}
	}

	// Extra tags
	for _, k := range search.GetVirtualTags() {
		uniqueMap[k] = struct{}{}
	}

	// Final response (sorted)
	resp := &tempopb.SearchTagsResponse{
		TagNames: make([]string, 0, len(uniqueMap)),
	}
	for k := range uniqueMap {
		resp.TagNames = append(resp.TagNames, k)
	}
	sort.Strings(resp.TagNames)

	return resp, nil
}

func (q *Querier) SearchTagValues(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.SearchTagValues")
	}

	// fetch response size limit for tag-values query
	tagValuesLimitBytes := q.limits.MaxBytesPerTagValuesQuery(userID)

	replicationSet, err := q.ring.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, errors.Wrap(err, "error finding ingesters in Querier.SearchTagValues")
	}

	// Get results from all ingesters
	lookupResults, err := q.forGivenIngesters(ctx, replicationSet, func(client tempopb.QuerierClient) (interface{}, error) {
		return client.SearchTagValues(ctx, req)
	})
	if err != nil {
		return nil, errors.Wrap(err, "error querying ingesters in Querier.SearchTagValues")
	}

	// Collect only unique values
	uniqueMap := map[string]struct{}{}
	for _, resp := range lookupResults {
		for _, res := range resp.response.(*tempopb.SearchTagValuesResponse).TagValues {
			uniqueMap[res] = struct{}{}
		}
	}

	// Extra values
	for _, v := range search.GetVirtualTagValues(req.TagName) {
		uniqueMap[v] = struct{}{}
	}

	if !util.MapSizeWithinLimit(uniqueMap, tagValuesLimitBytes) {
		return &tempopb.SearchTagValuesResponse{
			TagValues: []string{},
		}, nil
	}

	// Final response (sorted)
	resp := &tempopb.SearchTagValuesResponse{
		TagValues: make([]string, 0, len(uniqueMap)),
	}
	for k := range uniqueMap {
		resp.TagValues = append(resp.TagValues, k)
	}
	sort.Strings(resp.TagValues)

	return resp, nil
}

// SearchBlock searches the specified subset of the block for the passed tags.
func (q *Querier) SearchBlock(ctx context.Context, req *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	// todo: if the querier is not currently doing anything it should prefer handling the request itself to
	//  offloading to the external endpoint
	if len(q.cfg.SearchExternalEndpoints) != 0 {
		endpoint := q.cfg.SearchExternalEndpoints[rand.Intn(len(q.cfg.SearchExternalEndpoints))]
		return searchExternalEndpoint(ctx, endpoint, req)
	}

	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting org id in Querier.BackendSearch")
	}

	blockID, err := uuid.Parse(req.BlockID)
	if err != nil {
		return nil, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:       req.Version,
		TenantID:      tenantID,
		Encoding:      enc,
		IndexPageSize: req.IndexPageSize,
		TotalRecords:  req.TotalRecords,
		BlockID:       blockID,
		DataEncoding:  req.DataEncoding,
	}

	var searchErr error
	respMtx := sync.Mutex{}
	resp := &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	decoder, err := model.NewDecoder(req.DataEncoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewDecoder: %w", err)
	}

	err = q.store.IterateObjects(ctx, meta, int(req.StartPage), int(req.PagesToSearch), func(id common.ID, obj []byte) bool {
		respMtx.Lock()
		resp.Metrics.InspectedTraces++
		resp.Metrics.InspectedBytes += uint64(len(obj))
		respMtx.Unlock()

		metadata, err := decoder.Matches(id, obj, req.SearchReq)

		respMtx.Lock()
		defer respMtx.Unlock()
		if err != nil {
			searchErr = err
			return false
		}
		if metadata == nil {
			return false
		}

		resp.Traces = append(resp.Traces, metadata)
		return len(resp.Traces) >= int(req.SearchReq.Limit)
	})
	if err != nil {
		return nil, err
	}
	if searchErr != nil {
		return nil, searchErr
	}

	return resp, nil
}

func (q *Querier) postProcessSearchResults(req *tempopb.SearchRequest, rr []responseFromIngesters) *tempopb.SearchResponse {
	response := &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	traces := map[string]*tempopb.TraceSearchMetadata{}

	for _, r := range rr {
		sr := r.response.(*tempopb.SearchResponse)
		for _, t := range sr.Traces {
			// Just simply take first result for each trace
			if _, ok := traces[t.TraceID]; !ok {
				traces[t.TraceID] = t
			}
		}
		if sr.Metrics != nil {
			response.Metrics.InspectedBytes += sr.Metrics.InspectedBytes
			response.Metrics.InspectedTraces += sr.Metrics.InspectedTraces
			response.Metrics.InspectedBlocks += sr.Metrics.InspectedBlocks
			response.Metrics.SkippedBlocks += sr.Metrics.SkippedBlocks
		}
	}

	for _, t := range traces {
		if t.RootServiceName == "" {
			t.RootServiceName = trace.RootSpanNotYetReceivedText
		}
		response.Traces = append(response.Traces, t)
	}

	// Sort and limit results
	sort.Slice(response.Traces, func(i, j int) bool {
		return response.Traces[i].StartTimeUnixNano > response.Traces[j].StartTimeUnixNano
	})
	if req.Limit != 0 && int(req.Limit) < len(response.Traces) {
		response.Traces = response.Traces[:req.Limit]
	}

	return response
}

func searchExternalEndpoint(ctx context.Context, externalEndpoint string, searchReq *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	req, err := http.NewRequest(http.MethodGet, externalEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to make new request: %w", err)
	}
	req, err = api.BuildSearchBlockRequest(req, searchReq)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to build search block request: %w", err)
	}
	err = user.InjectOrgIDIntoHTTPRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to inject tenant id: %w", err)
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	metricEndpointDuration.WithLabelValues(externalEndpoint).Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to call http: %s, %w", externalEndpoint, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external endpoint returned %d, %s", resp.StatusCode, string(body))
	}
	var searchResp tempopb.SearchResponse
	err = jsonpb.Unmarshal(bytes.NewReader(body), &searchResp)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to unmarshal body: %s, %w", string(body), err)
	}
	return &searchResp, nil
}
