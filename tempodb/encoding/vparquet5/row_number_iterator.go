package vparquet5

import (
	"errors"

	pq "github.com/grafana/tempo/pkg/parquetquery"
)

type rowNumberIterator struct {
	rowNumbers []pq.RowNumber
}

var _ pq.Iterator = (*rowNumberIterator)(nil)

func (r *rowNumberIterator) String() string {
	return "rowNumberIterator()"
}

func (r *rowNumberIterator) Next() (*pq.IteratorResult, error) {
	if len(r.rowNumbers) == 0 {
		return nil, nil
	}

	res := &pq.IteratorResult{RowNumber: r.rowNumbers[0]}
	r.rowNumbers = r.rowNumbers[1:]
	return res, nil
}

func (r *rowNumberIterator) SeekTo(to pq.RowNumber, definitionLevel int) (*pq.IteratorResult, error) {
	var at *pq.IteratorResult

	for at, _ = r.Next(); r != nil && at != nil && pq.CompareRowNumbers(definitionLevel, at.RowNumber, to) < 0; {
		at, _ = r.Next()
	}

	return at, nil
}

func (r *rowNumberIterator) Close() {}

var _ pq.Iterator = (*virtualRowNumberIterator)(nil)

// virtualRowNumberIterator is an iterator reads a row count from a column in a parquet file and uses the numbers
// to calculate virtual row numbers on the given definitionLevel. This is used to iterate over spans in a trace without
// actually reading a column on span definition level.
type virtualRowNumberIterator struct {
	iter            pq.Iterator // iterator returning row count
	definitionLevel int

	at      pq.IteratorResult
	rowsMax int32
	rowsLef int32
}

func (v *virtualRowNumberIterator) String() string {
	return "virtualRowNumberIterator()"
}

func (v *virtualRowNumberIterator) Next() (*pq.IteratorResult, error) {
	if v.rowsLef == 0 {
		res, err := v.iter.Next()
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, nil
		}
		if len(res.Entries) != 1 {
			return nil, errors.New("expecting exactly one entry")
		}

		v.at.RowNumber = res.RowNumber
		v.setRows(res.Entries[0].Value.Int32())

		if v.rowsLef == 0 {
			return &v.at, nil
		}
	}

	v.rowsLef--
	v.at.RowNumber.Next(v.definitionLevel, v.definitionLevel, v.definitionLevel)

	return &v.at, nil
}

func (v *virtualRowNumberIterator) SeekTo(rowNumber pq.RowNumber, definitionLevel int) (*pq.IteratorResult, error) {
	scopeLevel := min(definitionLevel, v.definitionLevel-1)
	if pq.CompareRowNumbers(scopeLevel, rowNumber, v.at.RowNumber) != 0 {
		res, err := v.iter.SeekTo(rowNumber, scopeLevel)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, nil
		}
		if len(res.Entries) != 1 {
			return nil, errors.New("expecting exactly one entry")
		}

		v.at.RowNumber = res.RowNumber
		v.setRows(res.Entries[0].Value.Int32())

		if v.rowsLef == 0 {
			return &v.at, nil
		}
	} else {
		v.at.RowNumber = pq.TruncateRowNumber(scopeLevel, rowNumber)
		v.rowsLef = v.rowsMax
	}

	var seek int32
	if definitionLevel >= v.definitionLevel && rowNumber[v.definitionLevel] >= 0 {
		seek = rowNumber[v.definitionLevel] + 1
	}

	for seek > 0 && v.rowsLef > 0 {
		seek--
		v.rowsLef--
		v.at.RowNumber.Next(v.definitionLevel, v.definitionLevel, v.definitionLevel)
	}

	return &v.at, nil
}

func (v *virtualRowNumberIterator) Close() {
	v.iter.Close()
	v.setRows(0)
}

func (v *virtualRowNumberIterator) setRows(rows int32) {
	v.rowsMax = rows
	v.rowsLef = rows
}
