package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

type queryRangeSharder struct {
	next      http.RoundTripper
	reader    tempodb.Reader
	overrides overrides.Interface
	progress  searchProgressFactory
	cfg       QueryRangeSharderConfig
	logger    log.Logger
}

type QueryRangeSharderConfig struct {
	ConcurrentRequests    int           `yaml:"concurrent_jobs,omitempty"`
	TargetBytesPerRequest int           `yaml:"target_bytes_per_job,omitempty"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	QueryBackendAfter     time.Duration `yaml:"query_backend_after,omitempty"`
	Interval              time.Duration `yaml:"interval,omitempty"`
}

func newQueryRangeSharder(reader tempodb.Reader, o overrides.Interface, cfg QueryRangeSharderConfig, progress searchProgressFactory, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return &queryRangeSharder{
			next:      next,
			reader:    reader,
			overrides: o,
			cfg:       cfg,
			logger:    logger,

			progress: progress,
		}
	})
}

func (s queryRangeSharder) RoundTrip(r *http.Request) (*http.Response, error) {
	now := time.Now()

	var (
		isProm        bool
		err           error
		queryRangeReq *tempopb.QueryRangeRequest
		generatorReq  *queryRangeJob
		tenantID      string
	)

	// This route supports two flavors. (1) Prometheus-compatible (2) Tempo native
	// Remember which flavor this is and swap it so all
	// upstream calls are always Tempo native.
	if strings.Contains(r.RequestURI, api.PathPromQueryRange) {
		isProm = true
		// Swap upstream calls to the Tempo-native paths
		r.URL.Path = strings.ReplaceAll(r.URL.Path, api.PathPromQueryRange, api.PathMetricsQueryRange)
		r.RequestURI = strings.ReplaceAll(r.RequestURI, api.PathPromQueryRange, api.PathMetricsQueryRange)
		// Prom endpoint is called with 1-second precision timestamps
		// Round "now" to 1-second also.
		now = time.Unix(now.Unix(), 0)
	}

	queryRangeReq, err = api.ParseQueryRangeRequest(r)
	if err != nil {
		return s.respErrHandler(isProm, err)
	}

	_, err = traceql.Parse(queryRangeReq.Query)
	if err != nil {
		return s.respErrHandler(isProm, err)
	}

	ctx := r.Context()
	tenantID, err = user.ExtractOrgID(ctx)
	if err != nil {
		return s.respErrHandler(isProm, err)
	}
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.QueryRangeSharder")
	defer span.Finish()

	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	// calculate and enforce max search duration
	maxDuration := s.maxDuration(tenantID)
	if maxDuration != 0 && time.Duration(queryRangeReq.End-queryRangeReq.Start)*time.Second > maxDuration {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("range specified by start and end exceeds %s. received start=%d end=%d", maxDuration, queryRangeReq.Start, queryRangeReq.End))),
		}, nil
	}

	generatorReq = s.generatorRequest(*queryRangeReq)

	reqCh := make(chan *queryRangeJob, 1) // buffer of 1 allows us to insert ingestReq if it exists
	stopCh := make(chan struct{})
	defer close(stopCh)

	if generatorReq != nil {
		reqCh <- generatorReq
	}

	s.backendRequests(tenantID, queryRangeReq, now, reqCh, stopCh)

	wg := boundedwaitgroup.New(uint(s.cfg.ConcurrentRequests))
	c := traceql.QueryRangeCombiner{}
	mtx := sync.Mutex{}

	startedReqs := 0
	for job := range reqCh {
		if job.err != nil {
			return nil, fmt.Errorf("unexpected err building reqs: %w", job.err)
		}

		// When we hit capacity of boundedwaitgroup, wg.Add will block
		wg.Add(1)
		startedReqs++

		go func(job *queryRangeJob) {
			defer wg.Done()

			innerR := s.toUpstreamRequest(subCtx, job.req, r, tenantID)
			resp, err := s.next.RoundTrip(innerR)
			if err != nil {
				// context cancelled error happens when we exit early.
				// bail, and don't log and don't set this error.
				if errors.Is(err, context.Canceled) {
					_ = level.Debug(s.logger).Log("msg", "exiting early from sharded query", "url", innerR.RequestURI, "err", err)
					return
				}

				_ = level.Error(s.logger).Log("msg", "error executing sharded query", "url", innerR.RequestURI, "err", err)
				// progress.setError(err)
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK {
				/*statusCode := resp.StatusCode
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", err)
				}
				statusMsg := fmt.Sprintf("upstream: (%d) %s", statusCode, string(bytesMsg))
				progress.setStatus(statusCode, statusMsg)
				*/
				return
			}

			// successful query, read the body
			results := &tempopb.QueryRangeResponse{}
			err = (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(resp.Body, results)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", err)
				// progress.setError(err)
				return
			}

			mtx.Lock()
			defer mtx.Unlock()
			c.Combine(results.Series)
		}(job)
	}

	// wait for all goroutines running in wg to finish or cancelled
	wg.Wait()

	var bodyString string
	if isProm {
		promResp := s.convertToPromFormat(&tempopb.QueryRangeResponse{
			Series: c.Results(),
		})
		bytes, err := json.Marshal(promResp)
		if err != nil {
			return nil, err
		}
		bodyString = string(bytes)
	} else {
		m := &jsonpb.Marshaler{}
		bodyString, err = m.MarshalToString(&tempopb.QueryRangeResponse{
			Series: c.Results(),
		})
		if err != nil {
			return nil, err
		}
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptJSON},
		},
		Body:          io.NopCloser(strings.NewReader(bodyString)),
		ContentLength: int64(len([]byte(bodyString))),
	}

	return resp, nil
}

// blockMetas returns all relevant blockMetas given a start/end
func (s *queryRangeSharder) blockMetas(start, end int64, tenantID string) []*backend.BlockMeta {
	// reduce metas to those in the requested range
	allMetas := s.reader.BlockMetas(tenantID)
	metas := make([]*backend.BlockMeta, 0, len(allMetas)/50) // divide by 50 for luck
	for _, m := range allMetas {
		if m.StartTime.UnixNano() <= end &&
			m.EndTime.UnixNano() >= start {
			metas = append(metas, m)
		}
	}

	return metas
}

func (s *queryRangeSharder) backendRequests(tenantID string, searchReq *tempopb.QueryRangeRequest, now time.Time, reqCh chan *queryRangeJob, stopCh <-chan struct{}) {
	// request without start or end, search only in generator
	if searchReq.Start == 0 || searchReq.End == 0 {
		close(reqCh)
		return
	}

	// calculate duration (start and end) to search the backend blocks
	start, end := s.backendRange(now, searchReq.Start, searchReq.End, s.cfg.QueryBackendAfter)

	fmt.Println("Backend request range:",
		"reqStart", time.Unix(0, int64(searchReq.Start)), "reqEnd", time.Unix(0, int64(searchReq.End)),
		"start", time.Unix(0, int64(start)), "end", time.Unix(0, int64(end)),
	)

	// no need to search backend
	if start == end {
		close(reqCh)
		return
	}

	go func() {
		s.buildBackendRequests(tenantID, searchReq, start, end, reqCh, stopCh)
	}()
}

func (s *queryRangeSharder) buildBackendRequests(tenantID string, searchReq *tempopb.QueryRangeRequest, start, end uint64, reqCh chan *queryRangeJob, stopCh <-chan struct{}) {
	defer close(reqCh)

	timeWindowSize := uint64(s.cfg.Interval.Nanoseconds())

	for start < end {

		thisStart := start
		thisEnd := start + timeWindowSize
		if thisEnd > end {
			thisEnd = end
		}

		blocks := s.blockMetas(int64(thisStart), int64(thisEnd), tenantID)
		if len(blocks) == 0 {
			start = thisEnd
			continue
		}

		totalBlockSize := uint64(0)
		for _, b := range blocks {
			totalBlockSize += b.Size
		}

		shards := uint32(math.Ceil(float64(totalBlockSize) / float64(s.cfg.TargetBytesPerRequest)))

		for i := uint32(1); i <= shards; i++ {
			shardR := *searchReq
			shardR.Start = thisStart
			shardR.End = thisEnd
			shardR.Shard = i
			shardR.Of = shards

			select {
			case reqCh <- &queryRangeJob{req: shardR}:
			case <-stopCh:
				return
			}
		}

		start = thisEnd
	}
}

func (s *queryRangeSharder) backendRange(now time.Time, start, end uint64, queryBackendAfter time.Duration) (uint64, uint64) {
	backendAfter := uint64(now.Add(-queryBackendAfter).UnixNano())

	// adjust start/end if necessary. if the entire query range was inside backendAfter then
	// start will == end. This signals we don't need to query the backend.
	if end > backendAfter {
		end = backendAfter
	}
	if start > backendAfter {
		start = backendAfter
	}

	return start, end
}

func (s *queryRangeSharder) generatorRequest(searchReq tempopb.QueryRangeRequest) *queryRangeJob {
	now := time.Now()
	cutoff := uint64(now.Add(-s.cfg.QueryBackendAfter).UnixNano())

	// if there's no overlap between the query and ingester range just return nil
	if searchReq.End < cutoff {
		return nil
	}

	if searchReq.Start < cutoff {
		searchReq.Start = cutoff
	}

	// if ingester start == ingester end then we don't need to query it
	if searchReq.Start == searchReq.End {
		return nil
	}

	// Shard 0 indicates generator request
	searchReq.Shard = 0
	searchReq.Of = 0
	return &queryRangeJob{
		req: searchReq,
	}
}

func (s *queryRangeSharder) toUpstreamRequest(ctx context.Context, req tempopb.QueryRangeRequest, parent *http.Request, tenantID string) *http.Request {
	subR := parent.Clone(ctx)
	subR.Header.Set(user.OrgIDHeaderName, tenantID)
	subR = api.BuildQueryRangeRequest(subR, &req)
	subR.RequestURI = buildUpstreamRequestURI(parent.URL.Path, subR.URL.Query())
	return subR
}

// maxDuration returns the max search duration allowed for this tenant.
func (s *queryRangeSharder) maxDuration(tenantID string) time.Duration {
	// check overrides first, if no overrides then grab from our config
	maxDuration := s.overrides.MaxSearchDuration(tenantID)
	if maxDuration != 0 {
		return maxDuration
	}

	return s.cfg.MaxDuration
}

func (s *queryRangeSharder) convertToPromFormat(resp *tempopb.QueryRangeResponse) PromResponse {
	// Sort series alphabetically so they are stable in the UI
	sort.Slice(resp.Series, func(i, j int) bool {
		a := resp.Series[i].Labels
		b := resp.Series[j].Labels

		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k].Value.GetStringValue() < b[k].Value.GetStringValue() {
				return true
			}
		}
		return false
	})

	// Sort in increasing timestamp so that lines are drawn correctly
	for _, series := range resp.Series {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].TimestampMs < series.Samples[j].TimestampMs
		})
	}

	promResp := PromResponse{
		Status: "success",
		Data:   &PromData{ResultType: "matrix"},
	}

	for _, series := range resp.Series {
		promResult := PromResult{
			Metric: map[string]string{},
		}

		for _, label := range series.Labels {
			promResult.Metric[label.Key] = label.Value.GetStringValue()
		}

		promResult.Values = make([]interface{}, 0, len(series.Samples))
		for _, ts := range series.Samples {
			promResult.Values = append(promResult.Values, []interface{}{
				float64(ts.TimestampMs) / 1000.0,           // float for timestamp. assume it's seconds
				strconv.FormatFloat(ts.Value, 'f', -1, 64), // making assumptions about the float format returned from prom
			})
		}

		promResp.Data.Result = append(promResp.Data.Result, promResult)
	}

	return promResp
}

// returns an HTTP response or an error
func (s *queryRangeSharder) respErrHandler(isProm bool, err error) (*http.Response, error) {
	if isProm {
		resp := s.convertToPromError(err)
		_ = level.Debug(s.logger).Log("resp", fmt.Sprintf("%+v", resp))

		bytes, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal failed with: %w: %w", marshalErr, err)
		}
		bodyString := string(bytes)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				api.HeaderContentType: {api.HeaderAcceptJSON},
			},
			Body:          io.NopCloser(strings.NewReader(bodyString)),
			ContentLength: int64(len([]byte(bodyString))),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	}, nil
}

func (s *queryRangeSharder) convertToPromError(err error) PromResponse {
	return PromResponse{
		Status:    "error",
		ErrorType: "bad_data",
		Error:     err.Error(),
	}
}

type PromResponse struct {
	Status    string    `json:"status"`
	Data      *PromData `json:"data,omitempty"`
	ErrorType string    `json:"errorType,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type PromData struct {
	ResultType string       `json:"resultType"`
	Result     []PromResult `json:"result"`
}

type PromResult struct {
	Metric    map[string]string `json:"metric"`
	Values    []interface{}     `json:"values"`    // first entry is timestamp (float), second is value (string)
	Exemplars []interface{}     `json:"exemplars"` // first entry is timestamp (float), second is duration (float seconds), third is traceID (string)
}

type queryRangeJob struct {
	req tempopb.QueryRangeRequest
	err error
}
