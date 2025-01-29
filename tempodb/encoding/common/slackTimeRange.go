package common

import (
	"time"

	"github.com/grafana/tempo/pkg/dataquality"
)

func AdjustTimeRangeForSlack(tenantID string, ingestionSlack time.Duration, start, end uint32) (uint32, uint32) {
	now := time.Now()
	startOfRange := uint32(now.Add(-ingestionSlack).Unix())
	endOfRange := uint32(now.Add(ingestionSlack).Unix())

	warn := false
	if start < startOfRange {
		warn = true
		start = uint32(now.Unix())
	}
	if end > endOfRange || end < start {
		warn = true
		end = uint32(now.Unix())
	}

	if warn {
		dataquality.WarnOutsideIngestionSlack(tenantID)
	}

	return start, end
}
