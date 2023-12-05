package combiner

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

type TResponse interface {
	*tempopb.SearchResponse | *tempopb.SearchTagsResponse | *tempopb.SearchTagsV2Response | *tempopb.SearchTagValuesResponse | *tempopb.SearchTagValuesV2Response
}

type genericCombiner[R TResponse] struct {
	mu sync.Mutex

	final R

	combine func(body io.ReadCloser, final R) error
	result  func(R) (string, error)

	code          int
	statusMessage string
	err           error
}

func (c *genericCombiner[R]) AddRequest(res *http.Response, _ string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shouldQuit() {
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

	defer func() { _ = res.Body.Close() }()
	if err := c.combine(res.Body, c.final); err != nil {
		c.statusMessage = internalErrorMsg
		c.err = fmt.Errorf("error unmarshalling response body: %w", err)
		return c.err
	}

	return nil
}

func (c *genericCombiner[R]) Complete() (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bodyString, err := c.result(c.final)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: c.code,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptJSON},
		},
		Body:          io.NopCloser(strings.NewReader(bodyString)),
		ContentLength: int64(len([]byte(bodyString))),
	}, nil
}

func (c *genericCombiner[R]) StatusCode() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.code
}

func (c *genericCombiner[R]) ShouldQuit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.shouldQuit()
}

func (c *genericCombiner[R]) shouldQuit() bool {
	if c.err != nil {
		return true
	}

	if c.code/100 == 5 { // Bail on 5xx
		return true
	}

	// 2xx and 404 are OK
	return false
}
