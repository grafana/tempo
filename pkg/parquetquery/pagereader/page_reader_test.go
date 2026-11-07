package pagereader

import (
	"bytes"
	"math/rand/v2"
	"testing"

	pq "github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/encoding"
	"github.com/stretchr/testify/require"
)

// TestPageReaderLoad: a page's levels must agree with the chunk's max definition level.
func TestPageReaderLoad(t *testing.T) {
	type row struct{ ID int64 }
	var buf bytes.Buffer
	require.NoError(t, pq.Write(&buf, []row{{1}, {2}, {3}}))
	f, err := pq.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	cc := f.RowGroups()[0].ColumnChunks()[0]
	pages := cc.Pages()
	defer pages.Close()
	pg, err := pages.ReadPage()
	require.NoError(t, err)
	defer pq.Release(pg)

	var r PageReader
	r.slots.maxDef = 1 // ID is required: its pages have no definition levels
	require.ErrorContains(t, r.load(pg), "max definition level is 1")
	require.NoError(t, r.SetChunk(cc))
	require.NoError(t, r.load(pg))
	require.Equal(t, int(pg.NumValues()), r.slots.numSlots)
}

// TestSkipToRow compares skipToRow with a slot by slot walk.
func TestSkipToRow(t *testing.T) {
	rnd := rand.New(rand.NewPCG(3, 4))
	for _, repeated := range []bool{false, true} {
		for range 200 {
			numSlots := rnd.IntN(2_000)
			var defLevels, repLevels []byte
			const maxDef = 2
			for i := range numSlots {
				defLevels = append(defLevels, byte(rnd.IntN(maxDef+1)))
				if repeated {
					rep := byte(0)
					if i > 0 && rnd.IntN(5) != 0 {
						rep = byte(1 + rnd.IntN(2))
					}
					repLevels = append(repLevels, rep)
				}
			}
			slot := rnd.IntN(numSlots + 1)
			cur := int32(-1) // the row before the page
			index := 0
			for s := range slot {
				if !repeated || repLevels[s] == 0 {
					cur++
				}
				if defLevels[s] == maxDef {
					index++
				}
			}
			to := cur + int32(rnd.IntN(300))

			// Walk slot by slot to the first slot of row to.
			wantSlot, wantIndex, row := -1, index, cur
			for s := slot; s < numSlots; s++ {
				if !repeated || repLevels[s] == 0 {
					if row++; row == to {
						wantSlot = s
						break
					}
				}
				if defLevels[s] == maxDef {
					wantIndex++
				}
			}

			r := slotReader{maxDef: maxDef, defLevels: defLevels, repLevels: repLevels, numSlots: numSlots, slot: slot, index: index}
			moved := r.skipToRow(cur, to)
			if wantSlot < 0 || to <= cur+1 { // not on the page, or the next row: left to next
				require.False(t, moved)
				require.Equal(t, slot, r.slot)
				require.Equal(t, index, r.index)
				continue
			}
			require.True(t, moved, "slot %d cur %d to %d", slot, cur, to)
			require.Equal(t, wantSlot, r.slot)
			require.Equal(t, wantIndex, r.index)
		}
	}
}

// TestCheckValueCount: a page must hold one value (or dictionary index) per present slot.
func TestCheckValueCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		data encoding.Values
		want int
		ok   bool
	}{
		{"int64", encoding.Int64Values([]int64{1, 2, 3}), 3, true},
		{"int64 short", encoding.Int64Values([]int64{1, 2}), 3, false},
		{"int32 indexes long", encoding.Int32Values([]int32{0, 1, 0, 1}), 3, false},
		{"bytes", encoding.ByteArrayValues([]byte("abc"), []uint32{0, 1, 3}), 2, true},
		{"bytes short", encoding.ByteArrayValues([]byte("abc"), []uint32{0, 3}), 2, false},
		{"fixed", encoding.FixedLenByteArrayValues([]byte("abcd"), 2), 2, true},
		{"fixed ragged", encoding.FixedLenByteArrayValues([]byte("abcde"), 2), 2, false},
		{"bool last byte", encoding.BooleanValues([]byte{0xff, 0x01}), 9, true},
		{"bool extra byte", encoding.BooleanValues([]byte{0xff, 0x01, 0x00}), 9, false},
		{"bool short", encoding.BooleanValues([]byte{0xff}), 9, false},
	} {
		err := checkValueCount(tc.data, tc.want)
		if tc.ok {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
