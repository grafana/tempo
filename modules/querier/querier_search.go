package querier

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	urlParamMinDuration = "minDuration"
	urlParamMaxDuration = "maxDuration"
)

func (q *Querier) parseSearchRequest(r *http.Request) (*tempopb.SearchRequest, error) {
	req := &tempopb.SearchRequest{
		Tags:  map[string]string{},
		Limit: q.cfg.SearchDefaultResultLimit,
	}

	for k, v := range r.URL.Query() {
		// Skip reserved keywords
		if k == urlParamMinDuration || k == urlParamMaxDuration || k == api.URLParamLimit {
			continue
		}

		if len(v) > 0 && v[0] != "" {
			req.Tags[k] = v[0]
		}
	}

	if s, ok := extractQueryParam(r, urlParamMinDuration); ok {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid minDuration: %w", err)
		}
		req.MinDurationMs = uint32(dur.Milliseconds())
	}

	if s, ok := extractQueryParam(r, urlParamMaxDuration); ok {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid maxDuration: %w", err)
		}
		req.MaxDurationMs = uint32(dur.Milliseconds())

		if req.MinDurationMs != 0 && req.MinDurationMs > req.MaxDurationMs {
			return nil, errors.New("invalid maxDuration: must be greater than minDuration")
		}
	}

	if s, ok := extractQueryParam(r, api.URLParamLimit); ok {
		limit, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		if limit <= 0 {
			return nil, errors.New("invalid limit: must be a positive number")
		}
		req.Limit = uint32(limit)
	}

	if q.cfg.SearchMaxResultLimit != 0 && req.Limit > q.cfg.SearchMaxResultLimit {
		req.Limit = q.cfg.SearchMaxResultLimit
	}

	return req, nil
}

func extractQueryParam(r *http.Request, param string) (string, bool) {
	value := r.URL.Query().Get(param)
	return value, value != ""
}
