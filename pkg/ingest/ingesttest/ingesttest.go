// Package ingesttest provides test helpers for the ingest package.
// It is intended for use in tests only; no production code should import it.
package ingesttest

import (
	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/twmb/franz-go/pkg/kgo"
)

// NewClient wraps a *kgo.Client in an ingest.Client without starting the
// partition monitor goroutine.
func NewClient(client *kgo.Client) *ingest.Client {
	return ingest.NewBareClient(client, log.NewNopLogger())
}
