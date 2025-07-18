package bufferer

import (
	"time"

	"github.com/google/uuid"
)

const (
	maxBackoff       = 120 * time.Second
	maxFlushAttempts = 10
)

type completeOp struct {
	tenantID string
	blockID  uuid.UUID

	at       time.Time
	attempts int
	bo       time.Duration
}

func (o *completeOp) Key() string { return o.tenantID + "/" + o.blockID.String() }

func (o *completeOp) Priority() int64 { return -o.at.Unix() }

func (o *completeOp) backoff() time.Duration {
	o.bo *= 2
	if o.bo > maxBackoff {
		o.bo = maxBackoff
	}

	return o.bo
}
