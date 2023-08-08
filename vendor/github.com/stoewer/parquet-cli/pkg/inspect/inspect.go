package inspect

import (
	"sort"

	"github.com/parquet-go/parquet-go"
)

type Pagination struct {
	Limit  *int64
	Offset int64
}

func LeafColumns(file *parquet.File) []*parquet.Column {
	var leafs []*parquet.Column

	columns := []*parquet.Column{file.Root()}
	for len(columns) > 0 {
		col := columns[len(columns)-1]
		columns = columns[:len(columns)-1]

		if col.Leaf() {
			leafs = append(leafs, col)
		} else {
			columns = append(columns, col.Columns()...)
		}
	}

	sort.SliceStable(leafs, func(i, j int) bool { return leafs[i].Index() < leafs[j].Index() })
	return leafs
}
