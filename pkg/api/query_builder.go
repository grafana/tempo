package api

import (
	"net/url"
	"strings"
)

type queryBuilder struct {
	builder strings.Builder
}

// newQueryBuilder creates a new queryBuilder
func newQueryBuilder(init string) *queryBuilder {
	qb := &queryBuilder{
		builder: strings.Builder{},
	}

	qb.builder.WriteString(init)
	return qb
}

// jpe - test me

// addParam adds a new key/val pair to the query
// like https://cs.opensource.google/go/go/+/refs/tags/go1.22.5:src/net/url/url.go;l=972
func (qb *queryBuilder) addParam(key, value string) {
	if qb.builder.Len() > 0 {
		qb.builder.WriteByte('&')
	}
	qb.builder.WriteString(url.QueryEscape(key))
	qb.builder.WriteByte('=')
	qb.builder.WriteString(url.QueryEscape(value))
}

func (qb *queryBuilder) query() string {
	return qb.builder.String()
}

// Returns an url with query parameters
func BuildURLWithQueryParams(url string, queryParams map[string]string) string {
	qb := newQueryBuilder(url)
	for k, v := range queryParams {
		qb.addParam(k, v)
	}
	return qb.query()
}
