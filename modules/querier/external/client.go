package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/hedgedmetrics"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricEndpointDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Name:                            "querier_external_endpoint_duration_seconds",
		Help:                            "The duration of the external endpoints.",
		Buckets:                         prometheus.DefBuckets,
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{"endpoint"})
	metricExternalHedgedRequests = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "querier_external_endpoint_hedged_roundtrips_total",
			Help:      "Total number of hedged external requests.",
		},
	)
)

type Config struct {
	Backend string

	HTTPConfig     *HTTPConfig
	CloudRunConfig *CloudRunConfig

	HedgeRequestsAt   time.Duration
	HedgeRequestsUpTo int
}

type Client struct {
	httpClient    *http.Client
	endpoints     []string
	tokenProvider tokenProvider
}

type CloudRunConfig struct {
	Endpoints []string `yaml:"external_endpoints"`
	NoAuth    bool     // For testing
}

type HTTPConfig struct {
	Endpoints []string
}

type option func(client *Client) error

func withTokenProvider(provider tokenProvider) option {
	return func(client *Client) error {
		client.tokenProvider = provider
		return nil
	}
}

func NewClient(cfg *Config) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch cfg.Backend {
	case "":
		// For backwards compatibility, use unauthenticated http as the default.
		return newClientWithOpts(&commonConfig{
			endpoints:         cfg.HTTPConfig.Endpoints,
			hedgeRequestsAt:   cfg.HedgeRequestsAt,
			hedgeRequestsUpTo: cfg.HedgeRequestsUpTo,
		})
	case "google_cloud_run":
		provider, err := newGoogleProvider(ctx, cfg.CloudRunConfig.Endpoints, cfg.CloudRunConfig.NoAuth)
		if err != nil {
			return nil, err
		}
		return newClientWithOpts(&commonConfig{
			endpoints:         cfg.CloudRunConfig.Endpoints,
			hedgeRequestsAt:   cfg.HedgeRequestsAt,
			hedgeRequestsUpTo: cfg.HedgeRequestsUpTo,
		}, withTokenProvider(provider))

	case "aws_lambda":
		// TODO: implement RBAC for lambda. Here's one approach using API
		// gateway:
		// https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-use-lambda-authorizer.html#api-gateway-lambda-authorizer-flow
		return nil, fmt.Errorf("lambda external backend does not yet have RBAC support")
	default:
		return nil, fmt.Errorf("unknown external backend configured: %s", cfg.Backend)
	}
}

type commonConfig struct {
	hedgeRequestsAt   time.Duration
	hedgeRequestsUpTo int
	endpoints         []string
}

func newClientWithOpts(cfg *commonConfig, opts ...option) (*Client, error) {
	httpClient := http.DefaultClient
	if cfg.hedgeRequestsAt != 0 {
		var err error
		var stats *hedgedhttp.Stats
		httpClient, stats, err = hedgedhttp.NewClientAndStats(
			cfg.hedgeRequestsAt,
			cfg.hedgeRequestsUpTo,
			http.DefaultClient,
		)
		if err != nil {
			return nil, err
		}
		hedgedmetrics.Publish(stats, metricExternalHedgedRequests)
	}

	c := &Client{
		httpClient:    httpClient,
		endpoints:     cfg.endpoints,
		tokenProvider: &nilTokenProvider{},
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
	columnsJSON, err := json.Marshal(searchReq.DedicatedColumns)
	if err != nil {
		return nil, err
	}
	req, err = api.BuildSearchBlockRequest(req, searchReq, string(columnsJSON))
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to build search block request: %w", err)
	}
	req = api.AddServerlessParams(req, maxBytes)
	err = user.InjectOrgIDIntoHTTPRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("external endpoint failed to inject tenant id: %w", err)
	}
	token, err := s.tokenProvider.getToken(ctx, endpoint)
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
		return nil, fmt.Errorf("external endpoint failed to unmarshal body: %s: %w", string(body), err)
	}

	return &searchResp, nil
}
