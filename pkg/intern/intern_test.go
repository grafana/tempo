package intern

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestBasics(t *testing.T) {
	x := Get("abc123")
	y := Get("abc123")

	if x.Get() != y.Get() {
		t.Error("abc123 values differ")
	}
	if x.Get() != "abc123" {
		t.Error("x.Get not abc123")
	}
	if x != y {
		t.Error("abc123 pointers differ")
	}

	a1 := x.Get()
	a2 := y.Get()

	p1 := fmt.Sprintf("%08x\n", stringAddr(a1))
	p2 := fmt.Sprintf("%08x\n", stringAddr(a2))
	require.Equal(t, p1, p2)
}

func stringAddr(s string) uintptr {
	return uintptr(unsafe.Pointer(unsafe.StringData(s)))
}
