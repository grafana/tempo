package encoding

import (
	"bytes"
	"context"
	"io"
	"testing"

	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/stretchr/testify/assert"
)

func TestEmptyNestedIterator(t *testing.T) {
	r := bytes.NewReader([]byte{})
	i := NewIterator(r, v2.NewObjectReaderWriter())

	id, obj, err := i.Next(context.Background())
	assert.Nil(t, id)
	assert.Nil(t, obj)
	assert.Equal(t, io.EOF, err)
}
