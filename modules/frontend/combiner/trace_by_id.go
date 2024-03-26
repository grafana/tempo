package combiner

import (
	"bytes"
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
}

func NewTraceByID(maxBytes int) Combiner { // jpe test max bytes
	return &traceByIDCombiner{
		c:    trace.NewCombiner(maxBytes),
		code: http.StatusNotFound,
	}
}

// jpe - tenant injection?
func (c *traceByIDCombiner) AddResponse(res *http.Response, tenant string) error {
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
		return nil
	}

	// Read the body
	buff, err := io.ReadAll(res.Body)
	if err != nil {
		c.statusMessage = internalErrorMsg
		return fmt.Errorf("error reading response body: %w", err)
	}
	_ = res.Body.Close()

	// Unmarshal the body
	resp := &tempopb.TraceByIDResponse{}
	err = resp.Unmarshal(buff)
	if err != nil {
		c.statusMessage = internalErrorMsg
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	// inject tenant label as resource in trace jpe - does this work?
	//	InjectTenantResource(tenant, resp.Trace)

	// Consume the trace
	_, err = c.c.Consume(resp.Trace)
	return err
}

func (c *traceByIDCombiner) HTTPFinal() (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	statusCode := c.code
	traceResult, _ := c.c.Result()

	if traceResult == nil || statusCode != http.StatusOK {
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(c.statusMessage)),
			Header:     http.Header{},
		}, nil
	}

	buff, err := proto.Marshal(&tempopb.TraceByIDResponse{
		Trace: traceResult,
	}) // jpe - this is marshalling to proto. may be done for us
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
	return c.code
}

// ShouldQuit returns true if the response should be returned early.
func (c *traceByIDCombiner) ShouldQuit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.shouldQuit()
}

func (c *traceByIDCombiner) shouldQuit() bool {
	if c.code/100 == 5 { // Bail on 5xx
		return true
	}

	// test special case for 404
	if c.code == http.StatusNotFound {
		return false
	}

	// bail on other 400s
	if c.code/100 == 4 {
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
