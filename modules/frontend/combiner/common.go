package combiner

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/grafana/tempo/pkg/api"
	"google.golang.org/grpc/codes"
)

type TResponse interface {
	proto.Message
}

type PipelineResponse interface {
	HTTPResponse() *http.Response
	RequestData() any
}

type genericCombiner[T TResponse] struct {
	mu sync.Mutex

	current T // todo: state mgmt is mixed between the combiner and the various implementations. put it in one spot.

	new      func() T
	combine  func(partial T, final T, resp PipelineResponse) error
	finalize func(T) (T, error)
	diff     func(T) (T, error) // currently only implemented by the search combiner. required for streaming
	quit     func(T) bool

	// Used to determine the response code and when to stop
	httpStatusCode int
	httpRespBody   string
	// Used to marshal the response when using an HTTP Combiner, it doesn't affect for a GRPC combiner.
	httpMarshalingFormat string
}

// Init an HTTP combiner with default values. The marshaling format dictates how the response will be marshaled, including the Content-type header.
func initHTTPCombiner[T TResponse](c *genericCombiner[T], marshalingFormat string) {
	c.httpStatusCode = 200
	c.httpMarshalingFormat = marshalingFormat
}

// AddResponse is used to add a http response to the combiner.
func (c *genericCombiner[T]) AddResponse(r PipelineResponse) error {
	res := r.HTTPResponse()
	if res == nil {
		return nil
	}

	// todo: reevaluate this. should the caller owner the lifecycle of the http.response body?
	defer func() { _ = res.Body.Close() }()

	// test shouldQuit and set response all under the same lock. this prevents race conditions where
	// two responses can make it pass shouldQuit() with different results.
	shouldQuitAndSetResponse := func() (bool, error) {
		c.mu.Lock()
		defer c.mu.Unlock()

		if c.shouldQuit() {
			return true, nil
		}

		if res.StatusCode != http.StatusOK {
			bytesMsg, err := io.ReadAll(res.Body)
			if err != nil {
				return true, fmt.Errorf("error reading response body: %w", err)
			}
			c.httpRespBody = string(bytesMsg)
			c.httpStatusCode = res.StatusCode
			// don't return error. the error path is reserved for unexpected errors.
			// http pipeline errors should be returned through the final response. (Complete/TypedComplete/TypedDiff)
			return true, nil
		}

		return false, nil
	}

	if quit, err := shouldQuitAndSetResponse(); quit {
		return err
	}

	partial := c.new() // instantiating directly requires additional type constraints. this seemed cleaner: https://stackoverflow.com/questions/69573113/how-can-i-instantiate-a-non-nil-pointer-of-type-argument-with-generic-go

	switch res.Header.Get(api.HeaderContentType) {
	case api.HeaderAcceptProtobuf:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("error reading response body")
		}
		if err := proto.Unmarshal(b, partial); err != nil {
			return fmt.Errorf("error unmarshalling proto response body: %w", err)
		}
	default:
		// Assume json
		if err := jsonpb.Unmarshal(res.Body, partial); err != nil {
			return fmt.Errorf("error unmarshalling response body: %w", err)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// test again for should quit. it's possible that another response came in while we were unmarshalling that would make us quit.
	if c.shouldQuit() {
		return nil
	}

	c.httpStatusCode = res.StatusCode
	if err := c.combine(partial, c.current, r); err != nil {
		c.httpRespBody = internalErrorMsg
		return fmt.Errorf("error combining in combiner: %w", err)
	}

	return nil
}

// HTTPFinal, GRPCComplete, and GRPCDiff are all responsible for returning something
// usable in grpc streaming/http response.
func (c *genericCombiner[T]) HTTPFinal() (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	httpErr, _ := c.erroredResponse()
	if httpErr != nil {
		return httpErr, nil
	}

	final, err := c.finalize(c.current)
	if err != nil {
		return nil, err
	}

	var bodyString string
	if c.httpMarshalingFormat == api.HeaderAcceptProtobuf {
		buff, err := proto.Marshal(final)
		if err != nil {
			return nil, fmt.Errorf("error marshalling response body: %w", err)
		}
		bodyString = string(buff)
	} else {
		bodyString, err = new(jsonpb.Marshaler).MarshalToString(final)
		if err != nil {
			return nil, fmt.Errorf("error marshalling response body: %w", err)
		}
	}

	return &http.Response{
		StatusCode: 200,
		Header: http.Header{
			api.HeaderContentType: {c.httpMarshalingFormat},
		},
		Body:          io.NopCloser(strings.NewReader(bodyString)),
		ContentLength: int64(len([]byte(bodyString))),
	}, nil
}

func (c *genericCombiner[T]) GRPCFinal() (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var empty T
	_, grpcErr := c.erroredResponse()
	if grpcErr != nil {
		return empty, grpcErr
	}

	final, err := c.finalize(c.current)
	if err != nil {
		return empty, err
	}

	// clone the final response to prevent race conditions with marshalling this data
	finalClone := proto.Clone(final).(T)
	return finalClone, nil
}

func (c *genericCombiner[T]) GRPCDiff() (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var empty T
	_, grpcErr := c.erroredResponse()
	if grpcErr != nil {
		return empty, grpcErr
	}

	diff, err := c.diff(c.current)
	if err != nil {
		return empty, err
	}

	// clone the diff to prevent race conditions with marshalling this data
	diffClone := proto.Clone(diff)
	return diffClone.(T), nil
}

func (c *genericCombiner[T]) erroredResponse() (*http.Response, error) {
	if c.httpStatusCode == http.StatusOK {
		return nil, nil
	}

	// build grpc error and http response
	var grpcErr error
	if c.httpStatusCode/100 == 5 {
		grpcErr = status.Error(codes.Internal, c.httpRespBody)
	} else if c.httpStatusCode == http.StatusTooManyRequests {
		grpcErr = status.Error(codes.ResourceExhausted, c.httpRespBody)
	} else if c.httpStatusCode == http.StatusBadRequest {
		grpcErr = status.Error(codes.InvalidArgument, c.httpRespBody)
	} else {
		grpcErr = status.Error(codes.Unknown, c.httpRespBody)
	}
	httpResp := &http.Response{
		StatusCode: c.httpStatusCode,
		Status:     http.StatusText(c.httpStatusCode),
		Body:       io.NopCloser(strings.NewReader(c.httpRespBody)),
	}

	return httpResp, grpcErr
}

func (c *genericCombiner[R]) StatusCode() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.httpStatusCode
}

func (c *genericCombiner[R]) ShouldQuit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.shouldQuit()
}

func (c *genericCombiner[R]) shouldQuit() bool {
	if c.httpStatusCode/100 == 5 { // Bail on 5xx
		return true
	}

	if c.httpStatusCode/100 == 4 { // Bail on 4xx
		return true
	}

	if c.quit != nil && c.quit(c.current) {
		return true
	}

	// 2xx
	return false
}
