package ingest

import (
	"github.com/go-kit/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

// NewBareClient wraps an existing *kgo.Client in an ingest.Client without
// starting the partition monitor goroutine. The caller is responsible for
// the full lifecycle of the underlying kgo.Client.
func NewBareClient(client *kgo.Client, logger log.Logger) *Client {
	return &Client{
		Client: client,
		stopCh: make(chan struct{}),
		logger: logger,
	}
}
