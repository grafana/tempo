package querier

import (
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	urlParamMinDuration = "minDuration"
	urlParamMaxDuration = "maxDuration"
	urlParamLimit       = "limit"
)

func (q *Querier) parseSearchRequest(r *http.Request) (*tempopb.SearchRequest, error) {
	req := &tempopb.SearchRequest{
		Tags:  map[string]string{},
		Limit: q.cfg.SearchDefaultResultLimit,
	}

	for k, v := range r.URL.Query() {
		// Skip reserved keywords
		if k == urlParamMinDuration || k == urlParamMaxDuration || k == urlParamLimit {
			continue
		}

		if len(v) > 0 && v[0] != "" {
			req.Tags[k] = v[0]
		}
	}

	if s := r.URL.Query().Get(urlParamMinDuration); s != "" {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, err
		}
		req.MinDurationMs = uint32(dur.Milliseconds())
	}

	if s := r.URL.Query().Get(urlParamMaxDuration); s != "" {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, err
		}
		req.MaxDurationMs = uint32(dur.Milliseconds())
	}

	if s := r.URL.Query().Get(urlParamLimit); s != "" {
		limit, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		if limit > 0 {
			req.Limit = uint32(limit)
		}
	}

	if q.cfg.SearchMaxResultLimit != 0 && req.Limit > q.cfg.SearchMaxResultLimit {
		req.Limit = q.cfg.SearchMaxResultLimit
	}

	return req, nil
}
