// Package blocksharding splits a block into the read jobs a query is divided
// into. It is shared so the query frontend and anything reproducing its job
// split stay in agreement.
package blocksharding

import "github.com/grafana/tempo/v3/tempodb/backend"

// PagesPerRequest returns an integer value that indicates the number of pages
// that should be searched per query. This value is based on the target number of bytes
// 0 is returned if there is no valid answer
func PagesPerRequest(m *backend.BlockMeta, bytesPerRequest int) int {
	if m.Size_ == 0 || m.TotalRecords == 0 {
		return 0
	}
	// if the block is smaller than the bytesPerRequest, we can search the entire block
	if m.Size_ < uint64(bytesPerRequest) {
		return int(m.TotalRecords)
	}

	bytesPerPage := m.Size_ / uint64(m.TotalRecords)
	if bytesPerPage == 0 {
		return 0
	}

	pagesPerQuery := bytesPerRequest / int(bytesPerPage)
	if pagesPerQuery == 0 {
		pagesPerQuery = 1 // have to have at least 1 page per query
	}

	return pagesPerQuery
}
