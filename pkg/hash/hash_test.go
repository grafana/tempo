package hash

import (
	"bytes"
	"encoding/binary"
	"fmt"
	stdfnv "hash/fnv"
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigest_NewEqualsNewValue(t *testing.T) {
	d := New()
	_, _ = d.Write([]byte{0xa, 0xb, 0xc})
	_, _ = d.WriteString("abc")
	d.WriteUint64(123)
	want := d.Sum64()

	dv := NewValue()
	_, _ = dv.Write([]byte{0xa, 0xb, 0xc})
	_, _ = dv.WriteString("abc")
	dv.WriteUint64(123)
	require.Equal(t, want, dv.Sum64())
}

func TestDigest_WriteUint64_isLittleEndian(t *testing.T) {
	// make sure future changes don't change endianes
	d := New()
	d.WriteUint64(0x0123456789abcdef)
	got := d.Sum64()

	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], 0x0123456789abcdef)
	want := xxhash.Sum64(buf[:])

	require.Equal(t, want, got)
}

func TestSum_panicsWithGuidance(t *testing.T) {
	require.PanicsWithValue(t, "hash.Digest.Sum is not supported; use Sum64", func() {
		d := New()
		_ = d.Sum(nil)
	})
}

func TestSum64String_matchesSum64(t *testing.T) {
	s := "resource.service.name"
	require.Equal(t, Sum64([]byte(s)), Sum64String(s))
}

func TestTokenFunctions_returnExpectedShape(t *testing.T) {
	// These wrap stdlib fnv; the contract is that they don't allocate
	// surprising types and they're deterministic.
	id := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10}
	assert.Equal(t, TokenForTraceID(id), TokenForTraceID(id))
	assert.Equal(t, HashForTraceID(id), HashForTraceID(id))
	assert.Equal(t, TokenFor("tenant-x", id), TokenFor("tenant-x", id))
}

// Verify HashForTraceID doesn't collide within reasonable numbers, and estimate
// the hash collision rate if it does.
func TestHashForCollisionRate(t *testing.T) {
	var (
		n      = 1_000_000
		tokens = map[uint64]struct{}{}
		IDs    = make([][]byte, 0, n)
		r      = makeRnd()
	)

	for i := 0; i < n; i++ {
		traceID := randomBytes(r, 16)

		IDs = append(IDs, traceID)
		tokens[HashForTraceID(traceID)] = struct{}{}
	}

	// Ensure no duplicate trace IDs accidentally generated
	sort.Slice(IDs, func(i, j int) bool {
		return bytes.Compare(IDs[i], IDs[j]) == -1
	})
	for i := 1; i < len(IDs); i++ {
		if bytes.Equal(IDs[i-1], IDs[i]) {
			panic("same trace ID was generated, oops")
		}
	}

	missing := n - len(tokens)
	require.Zerof(t, missing, "missing 1 out of every %.2f trace ids", float32(n)/float32(missing))
}

var (
	benchTraceID16    = []byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}
	benchUserID10     = "tenant-abc"
	benchScopeName    = "go.opentelemetry.io/contrib/instrumentation/net/http"
	benchScopeVersion = "1.34.0"
	benchQueryStr     = `{ resource.service.name = "frontend" } | by(span.http.status_code)`
	benchTagName      = "resource.service.name"
	benchUint64       = uint64(0x0123456789abcdef)
)

func BenchmarkCompareDigestAndFNV1(b *testing.B) {
	sizes := []int{8, 16, 20, 30, 40, 60, 80}

	r := makeRnd()
	byteInputs := make([][]byte, len(sizes))
	strInputs := make([][]byte, len(sizes))
	for i, n := range sizes {
		byteInputs[i] = randomBytes(r, n)
		strInputs[i] = randomBytes(r, n)
	}

	b.Run("fnv1", func(b *testing.B) {
		for _, in := range byteInputs {
			b.Run(fmt.Sprintf("byte-%d", len(in)), func(b *testing.B) {
				b.Run("new", func(b *testing.B) {
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						h := stdfnv.New64()
						_, _ = h.Write(in)
						sink = h.Sum64()
					}
					_ = sink
				})

				b.Run("reuse", func(b *testing.B) {
					h := stdfnv.New64()
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						h.Reset()
						_, _ = h.Write(in)
						sink = h.Sum64()
					}
					_ = sink
				})
			})
		}

		for _, in := range strInputs {
			s := string(in)
			b.Run(fmt.Sprintf("str-%d", len(in)), func(b *testing.B) {
				b.Run("new", func(b *testing.B) {
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						h := stdfnv.New64()
						_, _ = h.Write([]byte(s))
						sink = h.Sum64()
					}
					_ = sink
				})

				b.Run("reuse", func(b *testing.B) {
					h := stdfnv.New64()
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						h.Reset()
						_, _ = h.Write([]byte(s))
						sink = h.Sum64()
					}
					_ = sink
				})
			})
		}

		// ilskey: distributor ILS grouping key — traceKey uint64 + scope name + scope version.
		b.Run("ilskey", func(b *testing.B) {
			b.Run("new", func(b *testing.B) {
				var buf [8]byte
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					h := stdfnv.New64()
					binary.LittleEndian.PutUint64(buf[:], benchUint64)
					_, _ = h.Write(buf[:])
					_, _ = h.Write([]byte(benchScopeName))
					_, _ = h.Write([]byte(benchScopeVersion))
					sink = h.Sum64()
				}
				_ = sink
			})

			b.Run("reuse", func(b *testing.B) {
				h := stdfnv.New64()
				var buf [8]byte
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					h.Reset()
					binary.LittleEndian.PutUint64(buf[:], benchUint64)
					_, _ = h.Write(buf[:])
					_, _ = h.Write([]byte(benchScopeName))
					_, _ = h.Write([]byte(benchScopeVersion))
					sink = h.Sum64()
				}
				_ = sink
			})
		})

		// queryrange: frontend query_range cache key — query string + start/end/step uint64s.
		b.Run("queryrange", func(b *testing.B) {
			b.Run("new", func(b *testing.B) {
				var buf [8]byte
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					h := stdfnv.New64()
					_, _ = h.Write([]byte(benchQueryStr))
					binary.LittleEndian.PutUint64(buf[:], 1000)
					_, _ = h.Write(buf[:])
					binary.LittleEndian.PutUint64(buf[:], 2000)
					_, _ = h.Write(buf[:])
					binary.LittleEndian.PutUint64(buf[:], 60)
					_, _ = h.Write(buf[:])
					sink = h.Sum64()
				}
				_ = sink
			})

			b.Run("reuse", func(b *testing.B) {
				h := stdfnv.New64()
				var buf [8]byte
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					h.Reset()
					_, _ = h.Write([]byte(benchQueryStr))
					binary.LittleEndian.PutUint64(buf[:], 1000)
					_, _ = h.Write(buf[:])
					binary.LittleEndian.PutUint64(buf[:], 2000)
					_, _ = h.Write(buf[:])
					binary.LittleEndian.PutUint64(buf[:], 60)
					_, _ = h.Write(buf[:])
					sink = h.Sum64()
				}
				_ = sink
			})
		})
	})

	b.Run("digest", func(b *testing.B) {
		for _, in := range byteInputs {
			b.Run(fmt.Sprintf("byte-%d", len(in)), func(b *testing.B) {
				b.Run("new", func(b *testing.B) {
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						d := New()
						_, _ = d.Write(in)
						sink = d.Sum64()
					}
					_ = sink
				})

				b.Run("reuse", func(b *testing.B) {
					d := New()
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						d.Reset()
						_, _ = d.Write(in)
						sink = d.Sum64()
					}
					_ = sink
				})
			})
		}

		for _, in := range strInputs {
			s := string(in)
			b.Run(fmt.Sprintf("str-%d", len(in)), func(b *testing.B) {
				b.Run("new", func(b *testing.B) {
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						d := New()
						_, _ = d.WriteString(s)
						sink = d.Sum64()
					}
					_ = sink
				})

				b.Run("reuse", func(b *testing.B) {
					d := New()
					b.ReportAllocs()
					var sink uint64
					for i := 0; i < b.N; i++ {
						d.Reset()
						_, _ = d.WriteString(s)
						sink = d.Sum64()
					}
					_ = sink
				})
			})
		}

		// ilskey: distributor ILS grouping key — traceKey uint64 + scope name + scope version.
		b.Run("ilskey", func(b *testing.B) {
			b.Run("new", func(b *testing.B) {
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					d := New()
					d.WriteUint64(benchUint64)
					_, _ = d.WriteString(benchScopeName)
					_, _ = d.WriteString(benchScopeVersion)
					sink = d.Sum64()
				}
				_ = sink
			})

			b.Run("reuse", func(b *testing.B) {
				d := New()
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					d.Reset()
					d.WriteUint64(benchUint64)
					_, _ = d.WriteString(benchScopeName)
					_, _ = d.WriteString(benchScopeVersion)
					sink = d.Sum64()
				}
				_ = sink
			})
		})

		// queryrange: frontend query_range cache key — query string + start/end/step uint64s.
		b.Run("queryrange", func(b *testing.B) {
			b.Run("new", func(b *testing.B) {
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					d := New()
					_, _ = d.WriteString(benchQueryStr)
					d.WriteUint64(1000)
					d.WriteUint64(2000)
					d.WriteUint64(60)
					sink = d.Sum64()
				}
				_ = sink
			})

			b.Run("reuse", func(b *testing.B) {
				d := New()
				b.ReportAllocs()
				var sink uint64
				for i := 0; i < b.N; i++ {
					d.Reset()
					_, _ = d.WriteString(benchQueryStr)
					d.WriteUint64(1000)
					d.WriteUint64(2000)
					d.WriteUint64(60)
					sink = d.Sum64()
				}
				_ = sink
			})
		})
	})
}

func BenchmarkHashForTraceID(b *testing.B) {
	b.ReportAllocs()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink = HashForTraceID(benchTraceID16)
	}
	_ = sink
}

func BenchmarkTokenFor_UserID(b *testing.B) {
	b.ReportAllocs()
	var sink uint32
	for i := 0; i < b.N; i++ {
		sink = TokenFor(benchUserID10, benchTraceID16[:4])
	}
	_ = sink
}

func BenchmarkSum64String_TagScope(b *testing.B) {
	b.ReportAllocs()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink = Sum64String(benchTagName)
	}
	_ = sink
}

func makeRnd() *rand.Rand {
	return rand.New(rand.NewPCG(1, 2))
}

func randomBytes(r *rand.Rand, n int) []byte {
	b := make([]byte, n)
	var buf [8]byte
	for i := 0; i < n; i += 8 {
		binary.LittleEndian.PutUint64(buf[:], r.Uint64())
		copy(b[i:], buf[:])
	}
	return b
}
