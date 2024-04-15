package frontend

import (
	"strconv"
	"strings"

	"github.com/grafana/tempo/tempodb/backend"
)

const (
	cacheKeyPrefixSearchJob       = "sj:"
	cacheKeyPrefixSearchTag       = "st:"
	cacheKeyPrefixSearchTagValues = "stv:"
)

func searchJobCacheKey(tenant string, queryHash uint64, start int64, end int64, meta *backend.BlockMeta, startPage, pagesToSearch int) string {
	return cacheKey(cacheKeyPrefixSearchJob, tenant, queryHash, start, end, meta, startPage, pagesToSearch)
}

// cacheKey returns a string that can be used as a cache key for a backend search job. if a valid key cannot be calculated
// it returns an empty string.
func cacheKey(prefix string, tenant string, queryHash uint64, start int64, end int64, meta *backend.BlockMeta, startPage, pagesToSearch int) string {
	// if the query hash is 0 we can't cache. this may occur if the user is using the old search api
	if queryHash == 0 {
		return ""
	}

	// unless the search range completely encapsulates the block range we can't cache. this is b/c different search ranges will return different results
	// for a given block unless the search range covers the entire block
	if !(meta.StartTime.Unix() > start &&
		meta.EndTime.Unix() < end) {
		return ""
	}

	sb := strings.Builder{}
	sb.Grow(len(prefix) +
		len(tenant) +
		1 + // :
		20 + // query hash
		1 + // :
		36 + // block id
		1 + // :
		3 + // start page
		1 + // :
		2) // 2 for pages to search
	sb.WriteString(prefix)
	sb.WriteString(tenant)
	sb.WriteString(":")
	sb.WriteString(strconv.FormatUint(queryHash, 10))
	sb.WriteString(":")
	sb.WriteString(meta.BlockID.String())
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(startPage))
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(pagesToSearch))

	return sb.String()
}
