package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStack(t *testing.T) {
	type testVal struct {
		val int
	}

	var (
		testStack Stack[*testVal]
		val       *testVal
	)

	assert.Len(t, testStack, 0)
	assert.True(t, testStack.IsEmpty(), "testStack expected to be empty")

	_, ok := testStack.Peek()
	assert.False(t, ok, "testStack.Peek() expected to return false")
	_, ok = testStack.Pop()
	assert.False(t, ok, "testStack.Pop() expected to return false")

	testStack.Push(&testVal{1})
	val, ok = testStack.Peek()
	require.True(t, ok)
	assert.Equal(t, &testVal{1}, val)

	testStack.Push(&testVal{2})
	val, ok = testStack.Peek()
	require.True(t, ok)
	assert.Equal(t, &testVal{2}, val)

	val, ok = testStack.Pop()
	require.True(t, ok)
	assert.Equal(t, &testVal{2}, val)

	val, ok = testStack.Pop()
	require.True(t, ok)
	assert.Equal(t, &testVal{1}, val)

	_, ok = testStack.Pop()
	assert.True(t, testStack.IsEmpty(), "testStack expected to be empty")
	assert.False(t, ok, "testStack.Pop() expected to return false")

	testStack.Push(&testVal{1})
	testStack.Reset()
	assert.True(t, testStack.IsEmpty(), "testStack expected to be empty")
	assert.Len(t, testStack, 0)
	assert.Greater(t, cap(testStack), 0)
}
