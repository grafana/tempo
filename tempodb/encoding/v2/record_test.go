package v2

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
)

func TestSortRecord(t *testing.T) {
	numRecords := 10
	expected := make([]common.Record, 0, numRecords)

	for i := 0; i < numRecords; i++ {
		r, err := makeRecord(t)
		if err != nil {
			assert.NoError(t, err, "unexpected error making trace record")
		}
		expected = append(expected, r)
	}

	common.SortRecords(expected)

	for i := range expected {
		if i == 0 {
			continue
		}

		idSmaller := expected[i-1].ID
		idLarger := expected[i].ID

		assert.NotEqual(t, 1, bytes.Compare(idSmaller, idLarger))
	}
}

func makeRecord(t *testing.T) (common.Record, error) {
	t.Helper()

	r := common.Record{
		Start:  rand.Uint64(),
		Length: rand.Uint32(),
	}

	_, err := rand.Read(r.ID)
	if err != nil {
		return common.Record{}, err
	}

	return r, nil
}
