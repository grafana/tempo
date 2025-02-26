package parquetquery

import (
	"strings"

	pq "github.com/parquet-go/parquet-go"
)

func GetColumnIndexByPath(pf *pq.File, s string) (index, depth, maxDef int) {
	colSelector := strings.Split(s, ".")
	n := pf.Root()
	for len(colSelector) > 0 {
		n = n.Column(colSelector[0])
		if n == nil {
			return -1, -1, -1
		}

		colSelector = colSelector[1:]
		depth++
	}

	return n.Index(), depth, n.MaxDefinitionLevel()
}

func HasColumn(pf *pq.File, s string) bool {
	index, _, _ := GetColumnIndexByPath(pf, s)
	return index >= 0
}
