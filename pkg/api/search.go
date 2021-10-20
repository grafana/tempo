package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	urlParamTags        = "tags"
	urlParamMinDuration = "minDuration"
	urlParamMaxDuration = "maxDuration"
	urlParamLimit       = "limit"
	urlParamStart       = "start"
	urlParamEnd         = "end"

	// todo(search): make configurable
	maxRange     = 1800 // 30 minutes
	defaultLimit = 20
)

type SearchRequestParser struct {
	SearchDefaultResultLimit uint32
	SearchMaxResultLimit     uint32
}

func (p *SearchRequestParser) Parse(r *http.Request) (*tempopb.SearchRequest, error) {
	req := &tempopb.SearchRequest{
		Tags:  map[string]string{},
		Limit: p.SearchDefaultResultLimit,
	}

	// Passing tags as individual query parameters is not supported anymore, clients should use the tags
	// query parameter instead. We still parse these tags since the initial Grafana implementation uses this.
	// As Grafana gets updated and/or versions using this get old we can remove this section.
	for k, v := range r.URL.Query() {
		// Skip reserved keywords
		if k == urlParamTags || k == urlParamMinDuration || k == urlParamMaxDuration || k == urlParamLimit {
			continue
		}

		if len(v) > 0 && v[0] != "" {
			req.Tags[k] = v[0]
		}
	}

	if encodedTags, ok := extractQueryParam(r, urlParamTags); ok {
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

	if s, ok := extractQueryParam(r, urlParamLimit); ok {
		limit, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		if limit <= 0 {
			return nil, errors.New("invalid limit: must be a positive number")
		}
		req.Limit = uint32(limit)
	}

	if p.SearchMaxResultLimit != 0 && req.Limit > p.SearchMaxResultLimit {
		req.Limit = p.SearchMaxResultLimit
	}

	return req, nil
}

func ParseBackendSearch(r *http.Request) (start, end int64, limit int, err error) {
	if s := r.URL.Query().Get(urlParamStart); s != "" {
		start, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(urlParamEnd); s != "" {
		end, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(urlParamLimit); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil {
			return
		}
	}

	if start == 0 || end == 0 {
		err = errors.New("please provide non-zero values for http parameters start and end")
		return
	}

	if limit == 0 {
		limit = defaultLimit
	}

	if end-start > maxRange {
		err = fmt.Errorf("range specified by start and end exceeds %d seconds. received start=%d end=%d", maxRange, start, end)
		return
	}
	if end <= start {
		err = fmt.Errorf("http parameter start must be before end. received start=%d end=%d", start, end)
		return
	}

	return
}

func extractQueryParam(r *http.Request, param string) (string, bool) {
	value := r.URL.Query().Get(param)
	return value, value != ""
}
