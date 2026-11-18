package api

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/tempopb"
)

const (
	presentationParam  = "include_service_graph_presentation"
	bypassServiceParam = "bypass_service"
	bypassRegexParam   = "bypass_service_regex"
)

// ParseServiceGraphPresentationRequest resolves response-only presentation rules.
// An omitted list inherits tenant defaults; a single empty value clears a list.
func ParseServiceGraphPresentationRequest(r *http.Request, services, patterns []string) (*tempopb.ServiceGraphPresentation, error) {
	q := r.URL.Query()
	flags, hasFlag := q[presentationParam]
	names, hasNames := q[bypassServiceParam]
	regexes, hasRegex := q[bypassRegexParam]
	if !hasFlag && !hasNames && !hasRegex {
		return nil, nil
	}
	if hasFlag {
		if len(flags) != 1 {
			return nil, fmt.Errorf("%s must occur once", presentationParam)
		}
		enabled, err := strconv.ParseBool(flags[0])
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", presentationParam, err)
		}
		if !enabled {
			if hasNames || hasRegex {
				return nil, fmt.Errorf("bypass parameters require presentation to be enabled")
			}
			return nil, nil
		}
	}
	if hasNames {
		services = names
	}
	if hasRegex {
		patterns = regexes
	}
	normalize := func(values []string) ([]string, error) {
		if len(values) == 1 && values[0] == "" {
			return []string{}, nil
		}
		for _, value := range values {
			if value == "" || !utf8.ValidString(value) {
				return nil, fmt.Errorf("presentation rules must contain nonempty UTF-8 strings")
			}
		}
		return append([]string{}, values...), nil
	}
	var err error
	services, err = normalize(services)
	if err != nil {
		return nil, err
	}
	patterns, err = normalize(patterns)
	if err != nil {
		return nil, err
	}
	if len(services)+len(patterns) > 128 {
		return nil, fmt.Errorf("presentation rules exceed limit of 128")
	}
	size := 0
	for _, value := range services {
		size += len(value)
	}
	for _, value := range patterns {
		size += len(value)
	}
	if size > 64*1024 {
		return nil, fmt.Errorf("presentation rules exceed limit of 65536 bytes")
	}
	for _, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("invalid bypass_service_regex: %w", err)
		}
	}
	return &tempopb.ServiceGraphPresentation{BypassServices: services, BypassServiceRegex: patterns}, nil
}

// WithoutServiceGraphPresentationParams keeps response-only rules out of backend requests and cache keys.
func WithoutServiceGraphPresentationParams(r *http.Request) *http.Request {
	q := r.URL.Query()
	keys := []string{presentationParam, bypassServiceParam, bypassRegexParam}
	if !slices.ContainsFunc(keys, q.Has) {
		return r
	}
	clone := r.Clone(r.Context())
	for _, key := range keys {
		q.Del(key)
	}
	clone.URL.RawQuery = q.Encode()
	return clone
}
