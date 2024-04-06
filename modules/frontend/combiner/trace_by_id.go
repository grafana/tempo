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
)

const (
	internalErrorMsg = "internal error"
)

type traceByIDCombiner struct {
	mu sync.Mutex

	c *trace.Combiner

	code          int
	statusMessage string
}

// NewTraceByID returns a trace id combiner. The trace by id combiner has a few different behaviors then the others
// - 404 is a valid response code. if all downstream jobs return 404 then it will return 404 with no body
// - translate tempopb.TraceByIDResponse to tempopb.Trace. all other combiners pass the same object through
// - runs the zipkin dedupe logic on the fully combined trace
// - encode the returned trace as either json or proto depending on the request
func NewTraceByID(maxBytes int) Combiner {
	return &traceByIDCombiner{
		c:    trace.NewCombiner(maxBytes),
		code: http.StatusNotFound,
	}
}

func (c *traceByIDCombiner) AddResponse(res *http.Response) error {
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

	// dedupe duplicate span ids
	deduper := newDeduper()
	traceResult = deduper.dedupe(traceResult)

	buff, err := proto.Marshal(&tempopb.TraceByIDResponse{
		Trace: traceResult,
	})
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
