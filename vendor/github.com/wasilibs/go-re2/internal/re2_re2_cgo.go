//go:build tinygo.wasm || re2_cgo

package internal

import (
	"fmt"
	"unsafe"

	"github.com/wasilibs/go-re2/internal/cre2"
)

type wasmPtr unsafe.Pointer

var nilWasmPtr = wasmPtr(nil)

type libre2ABI struct{}

func newABI() *libre2ABI {
	return &libre2ABI{}
}

func (abi *libre2ABI) startOperation(memorySize int) allocation {
	return allocation{}
}

func (abi *libre2ABI) endOperation(allocation) {
}

func newRE(abi *libre2ABI, pattern cString, opts CompileOptions) wasmPtr {
	opt := cre2.NewOpt()
	defer cre2.DeleteOpt(opt)
	cre2.OptSetMaxMem(opt, maxSize)
	cre2.OptSetLogErrors(opt, false)
	if opts.Longest {
		cre2.OptSetLongestMatch(opt, true)
	}
	if opts.Posix {
		cre2.OptSetPosixSyntax(opt, true)
	}
	if opts.CaseInsensitive {
		cre2.OptSetCaseSensitive(opt, false)
	}
	if opts.Latin1 {
		cre2.OptSetLatin1Encoding(opt)
	}
	return wasmPtr(cre2.New(pattern.ptr, pattern.length, opt))
}

func reError(abi *libre2ABI, rePtr wasmPtr) (int, string) {
	code := cre2.ErrorCode(unsafe.Pointer(rePtr))
	if code == 0 {
		return 0, ""
	}

	arg := cre2.CopyCString(cre2.ErrorArg(unsafe.Pointer(rePtr)))
	return code, arg
}

func numCapturingGroups(abi *libre2ABI, rePtr wasmPtr) int {
	return cre2.NumCapturingGroups(unsafe.Pointer(rePtr))
}

func deleteRE(_ *libre2ABI, rePtr wasmPtr) {
	cre2.Delete(unsafe.Pointer(rePtr))
}

func release(re *Regexp) {
	deleteRE(re.abi, re.ptr)
}

func match(re *Regexp, s cString, matchesPtr wasmPtr, nMatches uint32) bool {
	return cre2.Match(unsafe.Pointer(re.ptr), s.ptr,
		s.length, 0, s.length, 0, unsafe.Pointer(matchesPtr), int(nMatches))
}

func matchFrom(re *Regexp, s cString, startPos int, matchesPtr wasmPtr, nMatches uint32) bool {
	return cre2.Match(unsafe.Pointer(re.ptr), s.ptr,
		s.length, startPos, s.length, 0, unsafe.Pointer(matchesPtr), int(nMatches))
}

type allocation struct{}

func (*allocation) newCString(s string) cString {
	if len(s) == 0 {
		// TinyGo uses a null pointer to represent an empty string, but this
		// prevents us from distinguishing a match on the empty string vs no
		// match for subexpressions. So we replace with an empty-length slice
		// to a string that isn't null.
		s = "a"[0:0]
	}
	res := cString{
		ptr:    unsafe.Pointer(unsafe.StringData(s)),
		length: len(s),
	}
	return res
}

func (*allocation) newCStringFromBytes(s []byte) cString {
	res := cString{
		ptr:    unsafe.Pointer(unsafe.SliceData(s)),
		length: len(s),
	}
	return res
}

func (a *allocation) newCStringArray(n int) cStringArray {
	sz := int(unsafe.Sizeof(cString{})) * n
	ptr := cre2.Malloc(sz)
	for i := 0; i < sz; i++ {
		*(*byte)(unsafe.Add(ptr, i)) = 0
	}

	return cStringArray{ptr: wasmPtr(ptr)}
}

func (a *allocation) read(ptr wasmPtr, size int) []byte {
	return (*[1 << 30]byte)(unsafe.Pointer(ptr))[:size:size]
}

type cString struct {
	ptr    unsafe.Pointer
	length int
}

type cStringArray struct {
	ptr wasmPtr
}

func (a cStringArray) free() {
	cre2.Free(unsafe.Pointer(a.ptr))
}

func namedGroupsIter(_ *libre2ABI, rePtr wasmPtr) wasmPtr {
	return wasmPtr(cre2.NamedGroupsIterNew(unsafe.Pointer(rePtr)))
}

func namedGroupsIterNext(_ *libre2ABI, iterPtr wasmPtr) (string, int, bool) {
	var namePtr unsafe.Pointer
	var index int
	if !cre2.NamedGroupsIterNext(unsafe.Pointer(iterPtr), &namePtr, &index) {
		return "", 0, false
	}

	name := cre2.CopyCString(namePtr)
	return name, index, true
}

func namedGroupsIterDelete(_ *libre2ABI, iterPtr wasmPtr) {
	cre2.NamedGroupsIterDelete(unsafe.Pointer(iterPtr))
}

func readMatch(_ *allocation, cs cString, matchPtr wasmPtr, dstCap []int) []int {
	match := (*cString)(matchPtr)
	subStrPtr := match.ptr
	if subStrPtr == nil {
		return append(dstCap, -1, -1)
	}
	sIdx := uintptr(subStrPtr) - uintptr(cs.ptr)
	return append(dstCap, int(sIdx), int(sIdx+uintptr(match.length)))
}

func readMatches(alloc *allocation, cs cString, matchesPtr wasmPtr, n int, deliver func([]int) bool) {
	var dstCap [2]int

	for i := 0; i < n; i++ {
		dst := readMatch(alloc, cs, wasmPtr(unsafe.Add(unsafe.Pointer(matchesPtr), unsafe.Sizeof(cString{})*uintptr(i))), dstCap[:0])
		if !deliver(dst) {
			break
		}
	}
}

func newSet(_ *libre2ABI, opts CompileOptions) wasmPtr {
	opt := cre2.NewOpt()
	defer cre2.DeleteOpt(opt)
	cre2.OptSetMaxMem(opt, maxSize)
	cre2.OptSetLogErrors(opt, false)
	if opts.Longest {
		cre2.OptSetLongestMatch(opt, true)
	}
	if opts.Posix {
		cre2.OptSetPosixSyntax(opt, true)
	}
	if opts.CaseInsensitive {
		cre2.OptSetCaseSensitive(opt, false)
	}
	if opts.Latin1 {
		cre2.OptSetLatin1Encoding(opt)
	}
	return wasmPtr(cre2.NewSet(opt, 0))
}

func setAdd(set *Set, s cString) string {
	msgPtr := cre2.SetAdd(unsafe.Pointer(set.ptr), s.ptr, s.length)
	if msgPtr == nil {
		return unknownCompileError
	}
	msg := cre2.CopyCString(msgPtr)
	if msg != "ok" {
		cre2.Free(msgPtr)
		return fmt.Sprintf("error parsing regexp: %s", msg)
	}
	return ""
}

func setCompile(set *Set) int32 {
	return int32(cre2.SetCompile(unsafe.Pointer(set.ptr)))
}

func setMatch(set *Set, cs cString, matchedPtr wasmPtr, nMatch int) int {
	return cre2.SetMatch(unsafe.Pointer(set.ptr), cs.ptr, cs.length, unsafe.Pointer(matchedPtr), nMatch)
}

func deleteSet(_ *libre2ABI, setPtr wasmPtr) {
	cre2.SetDelete(unsafe.Pointer(setPtr))
}
