package v0

import (
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeRecord(t *testing.T) {
	expected, err := makeRecord(t)
	assert.NoError(t, err, "unexpected error making trace record")

	buff := make([]byte, recordLength)

	r := record{}
	marshalRecord(expected, buff)
	actual := r.UnmarshalRecord(buff)

	assert.Equal(t, expected, actual)
}

func makeRecord(t *testing.T) (*common.Record, error) {
	t.Helper()

	r := newRecord()
	_, err := rand.Read(r.ID)
	if err != nil {
		return nil, err
	}

	return r, nil
}
