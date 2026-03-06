package internal

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"sync/atomic"
)

const unknownCompileError = "unknown error compiling pattern"

type Set struct {
	ptr      wasmPtr
	abi      *libre2ABI
	opts     CompileOptions
	exprs    []string
	released uint32
}

func CompileSet(exprs []string, opts CompileOptions) (*Set, error) {
	abi := newABI()
	setPtr := newSet(abi, opts)
	set := &Set{
		ptr:   setPtr,
		abi:   abi,
		opts:  opts,
		exprs: exprs,
	}
	var estimatedMemorySize int
	for _, expr := range exprs {
		estimatedMemorySize += len(expr) + 2
	}

	alloc := abi.startOperation(estimatedMemorySize)
	defer abi.endOperation(alloc)

	for _, expr := range exprs {
		cs := alloc.newCString(expr)
		errMsg := setAdd(set, cs)
		if errMsg != "" {
			return nil, fmt.Errorf("%s", errMsg)
		}
	}
	setCompile(set)
	// Use func(interface{}) form for nottinygc compatibility.
	runtime.SetFinalizer(set, func(obj interface{}) {
		obj.(*Set).release()
	})
	return set, nil
}

func (set *Set) release() {
	if !atomic.CompareAndSwapUint32(&set.released, 0, 1) {
		return
	}
	deleteSet(set.abi, set.ptr)
}

// FindAllString finds all matches of the regular expressions in the Set against the input string.
// It returns a slice of indices of the matched patterns. If n >= 0, it returns at most n matches; otherwise, it returns all of them.
func (set *Set) FindAllString(s string, n int) []int {
	if n == 0 {
		return nil
	}
	if n < 0 {
		n = len(set.exprs)
	}
	alloc := set.abi.startOperation(len(s) + 8 + n*8)
	defer set.abi.endOperation(alloc)

	cs := alloc.newCString(s)

	var matches []int

	set.findAll(&alloc, cs, n, func(match int) {
		matches = append(matches, match)
	})
	return matches
}

// FindAll executes the Set against the input bytes. It returns a slice
// with the indices of the matched patterns. If n >= 0, it returns at most
// n matches; otherwise, it returns all of them.
func (set *Set) FindAll(b []byte, n int) []int {
	if n == 0 {
		return nil
	}
	if n < 0 {
		n = len(set.exprs)
	}
	alloc := set.abi.startOperation(len(b) + 8 + n*8)
	defer set.abi.endOperation(alloc)

	cs := alloc.newCStringFromBytes(b)

	var matches []int

	set.findAll(&alloc, cs, n, func(match int) {
		matches = append(matches, match)
	})

	return matches
}

func (set *Set) findAll(alloc *allocation, cs cString, n int, deliver func(match int)) {
	matchArr := alloc.newCStringArray(n)
	defer matchArr.free()

	matchedCount := setMatch(set, cs, matchArr.ptr, n)
	matches := alloc.read(matchArr.ptr, n*4)
	for i := 0; i < matchedCount && i < n; i++ {
		deliver(int(binary.LittleEndian.Uint32(matches[i*4:])))
	}

	runtime.KeepAlive(matchArr)
	runtime.KeepAlive(set) // don't allow finalizer to run during method
}
