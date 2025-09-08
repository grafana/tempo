package pipeline

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

type adjustStartEndWare struct {
	next AsyncRoundTripper[combiner.PipelineResponse]

	endNotAllowedFromNow time.Duration
	startWhenNotProvided time.Duration
	sendNanos            bool
}

// NewAdjustStartEndWare creates middleware that adjusts the "end" parameter of incoming requests. It is assumed that all
// request pipelines it exists in expect a "start" and "end" parameter. Traditionally some of these endpoints supported
// no start/end parameter which simply meant "search all recent data". This middleware changes that behavior by choosing
// a start/end.
// notAllowedFromNow is the duration from now that is not allowed to be used as an "end" parameter.
// minStart is the substitute start provided if no start is provided in the request.
func NewAdjustStartEndWare(notAllowedFromNow time.Duration, startWhenNotProvided time.Duration, sendNanos bool) AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &adjustStartEndWare{
			next:                 next,
			endNotAllowedFromNow: notAllowedFromNow,
			startWhenNotProvided: startWhenNotProvided,
			sendNanos:            sendNanos,
		}
	})
}

// jpe - test!
func (c adjustStartEndWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	httpReq := req.HTTPRequest()

	// look for end param and adjust if it's within notAllowed of now
	// consider that query_range uses unix epoch nanos while everything else uses seconds
	q := httpReq.URL.Query()

	endParam := q.Get("end")
	startParam := q.Get("start")

	if (startParam == "" && endParam != "") || (startParam != "" && endParam == "") {
		return NewBadRequest(errors.New("only one of start and end params provided. either provide both or neither to search recent data")), nil
	}

	var adjustStart, adjustEnd time.Time
	if startParam == "" && endParam == "" {
		// neither provided - set both
		adjustEnd = time.Now().Add(-c.endNotAllowedFromNow)
		adjustStart = time.Now().Add(-c.startWhenNotProvided)
	} else {
		// both provided - adjust end if needed
		reqEnd, err := strconv.ParseInt(endParam, 10, 64)
		if err != nil {
			return NewBadRequest(fmt.Errorf("error parsing end param: %w", err)), nil
		}

		// the metrics pipeline technically takes both nanos or seconds and uses this check as way to determine the difference.
		// this was copied from http.go : jpe - put it in a common place
		if len(endParam) > 10 {
			// adjust reqEnd from nanos to seconds, we'll do our cutoff in seconds, that's good enough
			reqEnd = reqEnd / int64(time.Second)
		}

		maxEnd := time.Now().Add(-c.endNotAllowedFromNow).Unix()
		if reqEnd > maxEnd {
			adjustEnd = time.Unix(maxEnd, 0)
		}
	}

	// now set what we need to
	if !adjustStart.IsZero() {
		q.Set("start", timeToParam(adjustStart, c.sendNanos))
	}
	if !adjustEnd.IsZero() {
		q.Set("end", timeToParam(adjustEnd, c.sendNanos))
	}
	httpReq.URL.RawQuery = q.Encode()

	return c.next.RoundTrip(req.CloneFromHTTPRequest(httpReq))
}

func timeToParam(t time.Time, nanos bool) string {
	if nanos {
		return strconv.FormatInt(t.UnixNano(), 10)
	}

	return strconv.FormatInt(t.Unix(), 10)
}
