package pipeline

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

// FederationInstance represents a single Tempo instance for federation.
type FederationInstance struct {
	Name     string
	Endpoint string
	OrgID    string
	Timeout  time.Duration
	Headers  map[string]string
}

// FederationConfig holds the configuration for federation.
type FederationConfig struct {
	Instances          []FederationInstance
	ConcurrentRequests int
}

// asyncFederationSharder is an AsyncMiddleware that fans out requests to multiple Tempo instances.
type asyncFederationSharder struct {
	next   AsyncRoundTripper[combiner.PipelineResponse]
	cfg    FederationConfig
	logger log.Logger
	client *http.Client
}

// NewAsyncFederationSharder creates a new federation sharder middleware.
func NewAsyncFederationSharder(cfg FederationConfig, logger log.Logger) AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		timeout := 30 * time.Second
		for _, inst := range cfg.Instances {
			if inst.Timeout > timeout {
				timeout = inst.Timeout
			}
		}

		level.Info(logger).Log("msg", "federation sharder initialized", "instances", len(cfg.Instances))

		return &asyncFederationSharder{
			next:   next,
			cfg:    cfg,
			logger: logger,
			client: &http.Client{
				Timeout: timeout,
			},
		}
	})
}

// RoundTrip fans out the request to all configured Tempo instances
func (f *asyncFederationSharder) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	ctx, span := tracer.Start(req.Context(), "federation.RoundTrip")
	defer span.End()

	level.Debug(f.logger).Log("msg", "federation fan-out starting", "instances", len(f.cfg.Instances), "path", req.HTTPRequest().URL.Path)

	// Build requests for all instances
	reqs, err := f.buildFederatedRequests(req)
	if err != nil {
		return nil, err
	}

	// Determine concurrency
	concurrentReqs := f.cfg.ConcurrentRequests
	if concurrentReqs <= 0 {
		concurrentReqs = len(f.cfg.Instances)
	}

	// Fan out to all instances using the same pattern as sharder
	return NewAsyncSharderFunc(ctx, concurrentReqs, len(reqs), func(i int) Request {
		return reqs[i]
	}, f.next), nil
}

// buildFederatedRequests creates a request for each Tempo instance.
func (f *asyncFederationSharder) buildFederatedRequests(parent Request) ([]Request, error) {
	httpReq := parent.HTTPRequest()

	// Read body once so we can replay it for each instance
	var bodyBytes []byte
	if httpReq.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(httpReq.Body)
		if err != nil {
			return nil, err
		}
		// Restore original body
		httpReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	reqs := make([]Request, len(f.cfg.Instances))
	for i, instance := range f.cfg.Instances {
		req, err := f.buildInstanceRequest(parent, instance, bodyBytes)
		if err != nil {
			level.Error(f.logger).Log("msg", "failed to build instance request", "instance", instance.Name, "err", err)
			continue
		}
		reqs[i] = req
	}

	return reqs, nil
}

// buildInstanceRequest creates a request for a specific Tempo instance.
func (f *asyncFederationSharder) buildInstanceRequest(parent Request, instance FederationInstance, bodyBytes []byte) (Request, error) {
	httpReq := parent.HTTPRequest()

	// Parse the instance endpoint
	instanceURL, err := url.Parse(instance.Endpoint)
	if err != nil {
		return nil, err
	}

	// Build the full URL by combining instance base URL with the request path
	fullURL := *instanceURL
	fullURL.Path = httpReq.URL.Path
	fullURL.RawQuery = httpReq.URL.RawQuery

	// Create body reader
	var body io.Reader
	if len(bodyBytes) > 0 {
		body = bytes.NewReader(bodyBytes)
	}

	// Create a new request for this instance
	newReq, err := http.NewRequestWithContext(parent.Context(), httpReq.Method, fullURL.String(), body)
	if err != nil {
		return nil, err
	}

	// Copy headers from original request
	for k, v := range httpReq.Header {
		newReq.Header[k] = v
	}

	// Override org ID if specified for this instance
	if instance.OrgID != "" {
		newReq.Header.Set("X-Scope-OrgID", instance.OrgID)
	}

	// Add any instance-specific headers
	for k, v := range instance.Headers {
		newReq.Header.Set(k, v)
	}

	level.Debug(f.logger).Log("msg", "built federated request", "instance", instance.Name, "url", fullURL.String())

	// Clone the pipeline request with the new HTTP request
	pipelineReq := parent.CloneFromHTTPRequest(newReq)
	pipelineReq.SetResponseData(instance.Name)

	return pipelineReq, nil
}

// NewHTTPRoundTripper creates a simple RoundTripper that executes HTTP requests.
// This is used as the RoundTripper in the pipeline when federation is enabled,
// replacing the normal querier connection with direct HTTP calls to Tempo instances.
func NewHTTPRoundTripper(timeout time.Duration, logger log.Logger) RoundTripper {
	client := &http.Client{
		Timeout: timeout,
	}

	return RoundTripperFunc(func(req Request) (*http.Response, error) {
		httpReq := req.HTTPRequest()

		// Get instance name from response data for logging
		instanceName := ""
		if data := req.ResponseData(); data != nil {
			instanceName, _ = data.(string)
		}

		level.Debug(logger).Log("msg", "executing federation HTTP request", "instance", instanceName, "url", httpReq.URL.String())
		resp, err := client.Do(httpReq)
		if err != nil {
			level.Warn(logger).Log("msg", "federation HTTP request failed", "instance", instanceName, "err", err)
			return nil, err
		}

		level.Debug(logger).Log("msg", "federation HTTP request complete", "instance", instanceName, "status", resp.StatusCode)
		return resp, nil
	})
}
