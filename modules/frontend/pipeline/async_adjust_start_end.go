package pipeline

import (
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/api"
)

type adjustStartEndWare struct {
	next AsyncRoundTripper[combiner.PipelineResponse]

	endBuffer time.Duration
	defStart  time.Duration
	sendNanos bool
}

// NewAdjustStartEndWare creates middleware that adjusts the "end" parameter of incoming requests. It is assumed that all
// request pipelines it exists in expect a "start" and "end" parameter. Traditionally some of these endpoints supported
// no start/end parameter which simply meant "search all recent data". This middleware changes that behavior by choosing
// a start/end.
// endBuffer is the duration from now that is not allowed to be used as an "end" parameter.
// defSstart is the substitute start provided if no start is provided in the request.

// Behavior:
//   - If the request does not include a "start" or "end", defaults are applied.
//   - The "end" parameter is always adjusted to avoid being too close to the current time.
//   - If "since" is present, it is ignored and replaced by the clamped "start" and "end" values.
//
// Parameters:
//   - defStart:   The default lookback duration (relative to "end") if no "start" is provided.
//   - endBuffer:  The minimum "gap" duration between now and the allowed "end" time.
//     This prevents queries from using "now" as an end time, which may hit incomplete data.
//   - sendNanos:  If true, times are encoded as Unix nanoseconds. Otherwise, Unix seconds are used.
func NewAdjustStartEndWare(defStart time.Duration, endBuffer time.Duration, sendNanos bool) AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &adjustStartEndWare{
			next:      next,
			defStart:  defStart,
			endBuffer: endBuffer,
			sendNanos: sendNanos,
		}
	})
}

func (c adjustStartEndWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	var (
		request    = req.HTTPRequest()
		query      = request.URL.Query()
		finalStart string
		finalEnd   string
	)

	start, end, err := api.ClampDateRangeReq(request, c.defStart, c.endBuffer)
	if err != nil {
		return NewBadRequest(fmt.Errorf("error parsing date range: %w", err)), nil
	}

	if c.sendNanos {
		finalStart = strconv.FormatInt(start.UnixNano(), 10)
		finalEnd = strconv.FormatInt(end.UnixNano(), 10)
	} else {
		finalStart = strconv.FormatInt(start.Unix(), 10)
		finalEnd = strconv.FormatInt(end.Unix(), 10)
	}

	query.Set("start", finalStart)
	query.Set("end", finalEnd)
	query.Del("since") // Since is overwritten to use the clamped start and end
	request.URL.RawQuery = query.Encode()

	return c.next.RoundTrip(req.CloneFromHTTPRequest(request))
}
