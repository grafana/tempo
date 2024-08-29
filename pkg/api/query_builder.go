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

// addParam adds a new key/val pair to the query
// like https://cs.opensource.google/go/go/+/refs/tags/go1.22.5:src/net/url/url.go;l=972
func (qb *queryBuilder) addParam(key, value string) {
	if qb.builder.Len() > 0 {
		qb.builder.WriteByte('&')
	}

	keyStr := url.QueryEscape(key)
	valueStr := url.QueryEscape(value)

	qb.builder.Grow(len(keyStr) + len(valueStr) + 1)

	qb.builder.WriteString(keyStr)
	qb.builder.WriteByte('=')
	qb.builder.WriteString(valueStr)
}

func (qb *queryBuilder) query() string {
	return qb.builder.String()
}
