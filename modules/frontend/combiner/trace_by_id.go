package combiner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

const (
	internalErrorMsg = "internal error"
	tenantLabel      = "tenant"
)

type traceByIDCombiner struct {
	mu sync.Mutex

	c *trace.Combiner

	code          int
	statusMessage string
	err           error
}

func NewTraceByID() Combiner {
	return &traceByIDCombiner{
		c:    trace.NewCombiner(0),
		code: http.StatusNotFound,
	}
}

func (c *traceByIDCombiner) AddRequest(res *http.Response, tenant string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shouldQuit() {
		return nil
	}

	if res.StatusCode == http.StatusNotFound {
		// 404s are not considered errors, so we don't need to do anything.
		return nil
	}
	c.code = res.StatusCode

	if res.StatusCode != http.StatusOK {
		bytesMsg, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}
		c.statusMessage = string(bytesMsg)
		return errors.New(c.statusMessage)
	}

	// Read the body
	buff, err := io.ReadAll(res.Body)
	if err != nil {
		c.statusMessage = internalErrorMsg
		c.err = fmt.Errorf("error reading response body: %w", err)
		return c.err
	}
	_ = res.Body.Close()

	// Unmarshal the body
	trace := &tempopb.Trace{}
	err = trace.Unmarshal(buff)
	if err != nil {
		c.statusMessage = internalErrorMsg
		c.err = fmt.Errorf("error unmarshalling response body: %w", err)
		return c.err
	}

	// inject tenant label as resource in trace
	InjectTenantResource(tenant, trace)

	// Consume the trace
	_, err = c.c.Consume(trace)
	return err
}

func (c *traceByIDCombiner) Complete() (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	statusCode := c.getStatusCode()
	traceResult, _ := c.c.Result()

	if traceResult == nil || statusCode != http.StatusOK {
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(c.statusMessage)),
			Header:     http.Header{},
		}, nil
	}

	buff, err := proto.Marshal(traceResult)
	if err != nil {
		return &http.Response{}, fmt.Errorf("error marshalling response to proto: %w", err)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptProtobuf},
		},
		Body:          io.NopCloser(bytes.NewReader(buff)),
		ContentLength: int64(len(buff)),
	}, nil
}

func (c *traceByIDCombiner) StatusCode() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getStatusCode()
}

func (c *traceByIDCombiner) getStatusCode() int {
	statusCode := c.code
	// Translate non-404s 4xx into 500s. If, for instance, we get a 400 back from an internal component
	// it means that we created a bad request. 400 should not be propagated back to the user b/c
	// the bad request was due to a bug on our side, so return 500 instead.
	if statusCode/100 == 4 && statusCode != http.StatusNotFound {
		statusCode = 500
	}

	return statusCode
}

// ShouldQuit returns true if the response should be returned early.
func (c *traceByIDCombiner) ShouldQuit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.shouldQuit()
}

func (c *traceByIDCombiner) shouldQuit() bool {
	if c.err != nil {
		return true
	}

	if c.getStatusCode()/100 == 5 { // Bail on 5xx
		return true
	}

	// 2xx and 404 are OK
	return false
}

// InjectTenantResource will add tenantLabel attribute into response to show which tenant the response came from
func InjectTenantResource(tenant string, t *tempopb.Trace) {
	if t == nil || t.Batches == nil {
		return
	}

	for _, b := range t.Batches {
		b.Resource.Attributes = append(b.Resource.Attributes, &v1.KeyValue{
			Key: tenantLabel,
			Value: &v1.AnyValue{
				Value: &v1.AnyValue_StringValue{
					StringValue: tenant,
				},
			},
		})
	}
}
