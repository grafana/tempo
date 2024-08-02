package api

import (
	"net/url"
	"strings"
)

type QueryBuilder struct {
	builder strings.Builder
}

// NewQueryBuilder creates a new queryBuilder
func NewQueryBuilder(init string) *QueryBuilder {
	qb := &QueryBuilder{
		builder: strings.Builder{},
	}

	qb.builder.WriteString(init)
	return qb
}

// jpe - test me

// addParam adds a new key/val pair to the query
// like https://cs.opensource.google/go/go/+/refs/tags/go1.22.5:src/net/url/url.go;l=972
func (qb *QueryBuilder) AddParam(key, value string) {
	if qb.builder.Len() > 0 {
		qb.builder.WriteByte('&')
	}
	qb.builder.WriteString(url.QueryEscape(key))
	qb.builder.WriteByte('=')
	qb.builder.WriteString(url.QueryEscape(value))
}

func (qb *QueryBuilder) Query() string {
	return qb.builder.String()
}
