package external

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/hedgedmetrics"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
	"golang.org/x/oauth2"
)

var (
	metricEndpointDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "querier_external_endpoint_duration_seconds",
		Help:      "The duration of the external endpoints.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"endpoint"})
	metricExternalHedgedRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "querier_external_endpoint_hedged_roundtrips_total",
			Help:      "Total number of hedged external requests. Registered as a gauge for code sanity. This is a counter.",
		},
	)
)

type Config struct {
	HedgeRequestsAt   time.Duration
	HedgeRequestsUpTo int
	Backend           string
	Endpoints         []string
}

type Client struct {
	httpClient    *http.Client
	endpoints     []string
	tokenProvider tokenProvider
}

type tokenProvider func(ctx context.Context, endpoint string) (*oauth2.Token, error)

func noToken(ctx context.Context, endpoint string) (*oauth2.Token, error) {
	// no-op
	return nil, nil
}

type option func(client *Client) error

func withTokenProvider(provider tokenProvider) option {
	return func(client *Client) error {
		client.tokenProvider = provider
		return nil
	}
}

func NewClient(cfg Config) (*Client, error) {
	switch cfg.Backend {
	case "":
		// For backwards compatibility, use unauthenticated http as the default.
		return newClientWithOpts(cfg)
	case "cloud-run":
		return newClientWithOpts(cfg, withTokenProvider(googleToken))
	case "lambda":
		// TODO: implement RBAC for lambda. Here's one approach using API
		// gateway:
		// https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-use-lambda-authorizer.html#api-gateway-lambda-authorizer-flow
		return nil, fmt.Errorf("lambda external backend does not yet have RBAC support")
	default:
		return nil, fmt.Errorf("unknown external backend configured: %s", cfg.Backend)
	}
}

func newClientWithOpts(cfg Config, opts ...option) (*Client, error) {
	httpClient := http.DefaultClient
	if cfg.HedgeRequestsAt != 0 {
		var err error
		var stats *hedgedhttp.Stats
		httpClient, stats, err = hedgedhttp.NewClientAndStats(
			cfg.HedgeRequestsAt,
			cfg.HedgeRequestsUpTo,
			http.DefaultClient,
		)

		if err != nil {
			return nil, err
		}
		hedgedmetrics.Publish(stats, metricExternalHedgedRequests)
	}

	c := &Client{
		httpClient:    httpClient,
		endpoints:     cfg.Endpoints,
		tokenProvider: noToken,
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (s *Client) Search(ctx context.Context, maxBytes int, searchReq *tempopb.SearchBlockRequest) (*tempopb.SearchResponse, error) {
	endpoint := s.endpoints[rand.Intn(len(s.endpoints))]
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to make new request: %w", err)
	}
	req, err = api.BuildSearchBlockRequest(req, searchReq)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to build search block request: %w", err)
	}
	req = api.AddServerlessParams(req, maxBytes)
	err = user.InjectOrgIDIntoHTTPRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to inject tenant id: %w", err)
	}
	token, err := s.tokenProvider(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to set auth header: %w", err)
	}
	if token != nil {
		token.SetAuthHeader(req)
	}

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	metricEndpointDuration.WithLabelValues(endpoint).Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to call http: %s, %w", endpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
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
