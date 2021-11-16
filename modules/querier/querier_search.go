package querier

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	urlParamTags        = "tags"
	urlParamMinDuration = "minDuration"
	urlParamMaxDuration = "maxDuration"
)

func (q *Querier) parseSearchRequest(r *http.Request) (*tempopb.SearchRequest, error) {
	req := &tempopb.SearchRequest{
		Tags:  map[string]string{},
		Limit: q.cfg.SearchDefaultResultLimit,
	}

	encodedTags, tagsFound := extractQueryParam(r, urlParamTags)

	if !tagsFound {
		// Passing tags as individual query parameters is not supported anymore, clients should use the tags
		// query parameter instead. We still parse these tags since the initial Grafana implementation uses this.
		// As Grafana gets updated and/or versions using this get old we can remove this section.
		for k, v := range r.URL.Query() {
			// Skip reserved keywords
			if k == urlParamTags || k == urlParamMinDuration || k == urlParamMaxDuration || k == api.URLParamLimit {
				continue
			}

			if len(v) > 0 && v[0] != "" {
				req.Tags[k] = v[0]
			}
		}
	}

	if tagsFound {
		decoder := logfmt.NewDecoder(strings.NewReader(encodedTags))

		for decoder.ScanRecord() {
			for decoder.ScanKeyval() {
				key := string(decoder.Key())
				if _, ok := req.Tags[key]; ok {
					return nil, fmt.Errorf("invalid tags: tag %s has been set twice", key)
				}
				req.Tags[key] = string(decoder.Value())
			}
		}

		if err := decoder.Err(); err != nil {
			if syntaxErr, ok := err.(*logfmt.SyntaxError); ok {
				return nil, fmt.Errorf("invalid tags: %s at pos %d", syntaxErr.Msg, syntaxErr.Pos)
			}
			return nil, fmt.Errorf("invalid tags: %w", err)
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
