package frontend

import (
	"strconv"
	"strings"

	"github.com/grafana/tempo/tempodb/backend"
)

const (
	cacheKeyPrefixSearchJob       = "sj:"
	cacheKeyPrefixSearchTag       = "st:"
	cacheKeyPrefixSearchTagValues = "st:"
)

func searchJobCacheKey(queryHash uint64, start int64, end int64, meta *backend.BlockMeta, startPage, pagesToSearch int) string {
	return cacheKey(cacheKeyPrefixSearchJob, queryHash, start, end, meta, startPage, pagesToSearch)
}

// cacheKey returns a string that can be used as a cache key for a backend search job. if a valid key cannot be calculated
// it returns an empty string.
func cacheKey(prefix string, queryHash uint64, start int64, end int64, meta *backend.BlockMeta, startPage, pagesToSearch int) string {
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
	sb.Grow(3 + 20 + 1 + 36 + 1 + 3 + 1 + 2) // 3 for prefix, 20 for query hash, 1 for :, 36 for block id, 1 for :, 3 for start page, 1 for :, 2 for pages to search
	sb.WriteString(prefix)                   // sj for search job. prefix prevents unexpected collisions and an easy way to version for future iterations
	sb.WriteString(strconv.FormatUint(queryHash, 10))
	sb.WriteString(":")
	sb.WriteString(meta.BlockID.String())
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(startPage))
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(pagesToSearch))

	return sb.String()
}
