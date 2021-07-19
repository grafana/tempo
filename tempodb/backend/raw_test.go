package backend

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWriter(t *testing.T) {
	m := &MockRawWriter{}
	w := NewWriter(m)
	ctx := context.Background()

	expected := []byte{0x01, 0x02, 0x03, 0x04}

	err := w.Write(ctx, "test", uuid.New(), "test", expected, false)
	assert.NoError(t, err)
	assert.Equal(t, expected, m.writeBuffer)

	_, err = w.Append(ctx, "test", uuid.New(), "test", nil, expected)
	assert.NoError(t, err)
	assert.Equal(t, expected, m.appendBuffer)

	err = w.CloseAppend(ctx, nil)
	assert.NoError(t, err)
	assert.True(t, m.closeAppendCalled)

	meta := NewBlockMeta("test", uuid.New(), "blerg", EncGZIP, "glarg")
	expected, _ = json.Marshal(meta)
	err = w.WriteBlockMeta(ctx, meta)
	assert.NoError(t, err)
	assert.Equal(t, expected, m.writeBuffer)
}

func TestReader(t *testing.T) {
	m := &MockRawReader{}
	r := NewReader(m)
	ctx := context.Background()

	expected := []byte{0x01, 0x02, 0x03, 0x04}
	m.R = expected
	actual, err := r.Read(ctx, "test", uuid.New(), "test", false)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	m.Range = expected
	err = r.ReadRange(ctx, "test", uuid.New(), "test", 10, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	expectedTenants := []string{"a", "b", "c"}
	m.L = expectedTenants
	actualTenants, err := r.Tenants(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedTenants, actualTenants)

	uuid1 := uuid.New()
	uuid2 := uuid.New()
	expectedBlocks := []uuid.UUID{uuid1, uuid2}
	m.L = []string{uuid1.String(), uuid2.String()}
	actualBlocks, err := r.Blocks(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, expectedBlocks, actualBlocks)

	// should fail b/c meta is not valid
	meta, err := r.BlockMeta(ctx, uuid.New(), "test")
	assert.Error(t, err)
	assert.Nil(t, meta)

	expectedMeta := NewBlockMeta("test", uuid.New(), "blerg", EncGZIP, "glarg")
	m.R, _ = json.Marshal(expectedMeta)
	meta, err = r.BlockMeta(ctx, uuid.New(), "test")
	assert.NoError(t, err)
	e, _ := json.Marshal(expectedMeta)
	a, _ := json.Marshal(meta)
	assert.Equal(t, e, a)
}

func TestKeyPathForBlock(t *testing.T) {
	b := uuid.New()
	tid := "test"
	keypath := KeyPathForBlock(b, tid)

	assert.Equal(t, KeyPath([]string{tid, b.String()}), keypath)
}
