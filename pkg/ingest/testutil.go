package ingest

import (
	"github.com/go-kit/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

// NewClientForTesting wraps a *kgo.Client in an ingest.Client without starting
// the partition monitor goroutine. For use in unit tests only.
func NewClientForTesting(client *kgo.Client) *Client {
	return &Client{
		Client: client,
		stopCh: make(chan struct{}),
		logger: log.NewNopLogger(),
	}
}
