package blocklist

import (
	"time"
)

type tenantOp struct {
	at       time.Time // When to execute
	attempts uint
	tenantID string
}

func (o *tenantOp) Key() string {
	return o.tenantID
}

// Priority orders entries in the queue. The larger the number the higher the priority, so inverted here to
// prioritize entries with earliest timestamps.
func (o *tenantOp) Priority() int64 {
	return -o.at.Unix()
}
