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

func PathToDisplayName(path []string) string {
	l := len(path)
	if l > 3 {
		if path[l-2] == "list" && path[l-1] == "element" {
			return path[l-3]
		} else if path[l-2] == "key_value" && (path[l-1] == "key" || path[l-1] == "value") {
			return path[l-3] + "." + path[l-1]
		}
	}
	return path[l-1]
}
