// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package parseutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/parseutils"

import (
	"net/url"
	"strconv"
	"strings"

	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
)

const (
	// replace once conventions includes these
	AttributeURLUserInfo = "url.user_info"
	AttributeURLUsername = "url.username"
	AttributeURLPassword = "url.password"
)

// parseURI takes an absolute or relative uri and returns the parsed values.
func ParseURI(value string, semconvCompliant bool) (map[string]any, error) {
	m := make(map[string]any)

	if strings.HasPrefix(value, "?") {
		// remove the query string '?' prefix before parsing
		v, err := url.ParseQuery(value[1:])
		if err != nil {
			return nil, err
		}
		return queryToMap(v, m), nil
	}

	var x *url.URL
	var err error
	var mappingFn func(*url.URL, map[string]any) (map[string]any, error)

	if semconvCompliant {
		mappingFn = urlToSemconvMap
		x, err = url.Parse(value)
		if err != nil {
			return nil, err
		}
	} else {
		x, err = url.ParseRequestURI(value)
		if err != nil {
			return nil, err
		}

		mappingFn = urlToMap
	}
	return mappingFn(x, m)
}

// urlToMap converts a url.URL to a map, excludes any values that are not set.
func urlToSemconvMap(parsedURI *url.URL, m map[string]any) (map[string]any, error) {
	m[semconv.AttributeURLOriginal] = parsedURI.String()
	m[semconv.AttributeURLDomain] = parsedURI.Hostname()
	m[semconv.AttributeURLScheme] = parsedURI.Scheme
	m[semconv.AttributeURLPath] = parsedURI.Path

	if portString := parsedURI.Port(); len(portString) > 0 {
		port, err := strconv.Atoi(portString)
		if err != nil {
			return nil, err
		}
		m[semconv.AttributeURLPort] = port
	}

	if fragment := parsedURI.Fragment; len(fragment) > 0 {
		m[semconv.AttributeURLFragment] = fragment
	}

	if parsedURI.User != nil {
		m[AttributeURLUserInfo] = parsedURI.User.String()

		if username := parsedURI.User.Username(); len(username) > 0 {
			m[AttributeURLUsername] = username
		}

		if pwd, isSet := parsedURI.User.Password(); isSet {
			m[AttributeURLPassword] = pwd
		}
	}

	if query := parsedURI.RawQuery; len(query) > 0 {
		m[semconv.AttributeURLQuery] = query
	}

	if periodIdx := strings.LastIndex(parsedURI.Path, "."); periodIdx != -1 {
		if periodIdx < len(parsedURI.Path)-1 {
			m[semconv.AttributeURLExtension] = parsedURI.Path[periodIdx+1:]
		}
	}

	return m, nil
}

// urlToMap converts a url.URL to a map, excludes any values that are not set.
func urlToMap(p *url.URL, m map[string]any) (map[string]any, error) {
	scheme := p.Scheme
	if scheme != "" {
		m["scheme"] = scheme
	}

	user := p.User.Username()
	if user != "" {
		m["user"] = user
	}

	host := p.Hostname()
	if host != "" {
		m["host"] = host
	}

	port := p.Port()
	if port != "" {
		m["port"] = port
	}

	path := p.EscapedPath()
	if path != "" {
		m["path"] = path
	}

	return queryToMap(p.Query(), m), nil
}

// queryToMap converts a query string url.Values to a map.
func queryToMap(query url.Values, m map[string]any) map[string]any {
	// no-op if query is empty, do not create the key m["query"]
	if len(query) == 0 {
		return m
	}

	/* 'parameter' will represent url.Values
	map[string]any{
		"parameter-a": []any{
			"a",
			"b",
		},
		"parameter-b": []any{
			"x",
			"y",
		},
	}
	*/
	parameters := map[string]any{}
	for param, values := range query {
		parameters[param] = queryParamValuesToMap(values)
	}
	m["query"] = parameters
	return m
}

// queryParamValuesToMap takes query string parameter values and
// returns an []interface populated with the values
func queryParamValuesToMap(values []string) []any {
	v := make([]any, len(values))
	for i, value := range values {
		v[i] = value
	}
	return v
}
