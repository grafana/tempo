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

// JobBytes returns the bytes of a block that a job covers, assuming row groups of equal size, like
// PagesPerRequest does when the query frontend creates the jobs. startPage, pages and totalRecords are
// counted in parquet row groups, despite their names. The last job of a block may cover fewer row groups
// than requested and a job past the end of the block covers nothing. 0 is returned if totalRecords is 0.
func JobBytes(blockSize uint64, startPage, pages, totalRecords uint32) uint64 {
	if totalRecords == 0 || startPage >= totalRecords {
		return 0
	}

	rowGroups := uint64(min(pages, totalRecords-startPage))

	// split to not overflow for big blocks
	total := uint64(totalRecords)
	return blockSize/total*rowGroups + blockSize%total*rowGroups/total
}
