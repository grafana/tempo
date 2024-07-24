package combiner

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

type traceByIDCombinerV2 struct {
	mu sync.Mutex

	c           *trace.Combiner
	contentType string

	code          int
	statusMessage string
}

// NewTraceByID returns a trace id combiner. The trace by id combiner has a few different behaviors then the others
// - 404 is a valid response code. if all downstream jobs return 404 then it will return 404 with no body
// - translate tempopb.TraceByIDResponse to tempopb.Trace. all other combiners pass the same object through
// - runs the zipkin dedupe logic on the fully combined trace
// - encode the returned trace as either json or proto depending on the request
func NewTraceByIDV2(maxBytes int, contentType string) Combiner {
	return &traceByIDCombinerV2{
		c:           trace.NewCombiner(maxBytes),
		code:        http.StatusNotFound,
		contentType: contentType,
	}
}

func (c *traceByIDCombinerV2) AddResponse(r PipelineResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shouldQuit() {
		return nil
	}

	res := r.HTTPResponse()
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

func (c *traceByIDCombinerV2) HTTPFinal() (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	statusCode := c.code
	traceResult, _ := c.c.Result()

	if statusCode != http.StatusOK {
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(c.statusMessage)),
			Header:     http.Header{},
		}, nil
	}

	// if we have no trace result just substitute and return an empty trace
	if traceResult == nil {
		traceResult = &tempopb.Trace{}
	}

	// dedupe duplicate span ids
	deduper := newDeduper()
	traceResult = deduper.dedupe(traceResult)

	resp := &tempopb.TraceByIDResponse{Trace: traceResult}

	// marshal in the requested format
	var buff []byte
	var err error

	if c.contentType == api.HeaderAcceptProtobuf {
		buff, err = proto.Marshal(resp)
	} else {
		var jsonStr string

		marshaler := &jsonpb.Marshaler{}
		jsonStr, err = marshaler.MarshalToString(resp)
		buff = []byte(jsonStr)
	}
	if err != nil {
		return &http.Response{}, fmt.Errorf("error marshalling response: %w content type: %s", err, c.contentType)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {c.contentType},
		},
		Body:          io.NopCloser(bytes.NewReader(buff)),
		ContentLength: int64(len(buff)),
	}, nil
}

func (c *traceByIDCombinerV2) StatusCode() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.code
}

// ShouldQuit returns true if the response should be returned early.
func (c *traceByIDCombinerV2) ShouldQuit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.shouldQuit()
}

func (c *traceByIDCombinerV2) shouldQuit() bool {
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
