package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

// Wiresmith CustomMarshaler implementations for the (wiresmith.options.customtype)
// fields in v1/v1.proto (BlockMeta.block_id and BlockMeta.dedicated_columns).
// They delegate to the existing gogo-style Marshal/MarshalTo/Unmarshal/Size
// methods, so the wire bytes are identical to the old gogofaster output.

func (u *UUID) SizeWiresmith() int {
	if u == nil {
		return 0
	}
	return u.Size()
}

func (u *UUID) MarshalWiresmith(buf []byte) (int, error) {
	return u.MarshalTo(buf)
}

func (u *UUID) UnmarshalWiresmith(buf []byte) error {
	return u.Unmarshal(buf)
}

func (u *UUID) EqualWiresmith(other any) bool {
	switch o := other.(type) {
	case UUID:
		return *u == o
	case *UUID:
		return o != nil && *u == *o
	default:
		return false
	}
}

func (u *UUID) CompareWiresmith(other any) int {
	switch o := other.(type) {
	case UUID:
		return bytes.Compare(u[:], o[:])
	case *UUID:
		if o == nil {
			return 1
		}
		return bytes.Compare(u[:], o[:])
	default:
		return -1
	}
}

func (dcs *DedicatedColumns) SizeWiresmith() int {
	if dcs == nil {
		return 0
	}
	return dcs.Size()
}

func (dcs *DedicatedColumns) MarshalWiresmith(buf []byte) (int, error) {
	return dcs.MarshalTo(buf)
}

func (dcs *DedicatedColumns) UnmarshalWiresmith(buf []byte) error {
	return dcs.Unmarshal(buf)
}

func (dcs *DedicatedColumns) EqualWiresmith(other any) bool {
	o, ok := dedicatedColumnsValue(other)
	if !ok {
		return false
	}
	if len(*dcs) != len(o) {
		return false
	}
	for i := range *dcs {
		a, b := (*dcs)[i], o[i]
		if a.Scope != b.Scope || a.Name != b.Name || a.Type != b.Type {
			return false
		}
		if !slices.Equal(a.Options, b.Options) {
			return false
		}
	}
	return true
}

func (dcs *DedicatedColumns) CompareWiresmith(other any) int {
	o, ok := dedicatedColumnsValue(other)
	if !ok {
		return -1
	}
	// Ordering is only used for a stable total order; compare the encoded form.
	a, err := dcs.Marshal()
	if err != nil {
		return -1
	}
	b, err := o.Marshal()
	if err != nil {
		return 1
	}
	return bytes.Compare(a, b)
}

func dedicatedColumnsValue(other any) (DedicatedColumns, bool) {
	switch o := other.(type) {
	case DedicatedColumns:
		return o, true
	case *DedicatedColumns:
		if o == nil {
			return nil, false
		}
		return *o, true
	default:
		return nil, false
	}
}

// MarshalJSON flattens the BlockMeta fields into the top-level object,
// reproducing the JSON shape gogo's (gogoproto.embed) produced. Stored
// meta.compacted.json files and the tenant index "compacted" entries depend
// on this shape. The inverse lives in block_meta.go (UnmarshalJSON).
func (b *CompactedBlockMeta) MarshalJSON() ([]byte, error) {
	bm, err := json.Marshal(&b.BlockMeta)
	if err != nil {
		return nil, err
	}
	ct, err := json.Marshal(b.CompactedTime)
	if err != nil {
		return nil, err
	}
	if len(bm) < 2 || bm[len(bm)-1] != '}' {
		return nil, fmt.Errorf("unexpected BlockMeta JSON shape: %s", bm)
	}
	out := make([]byte, 0, len(bm)+len(ct)+len(`,"compactedTime":`))
	out = append(out, bm[:len(bm)-1]...)
	if len(bm) > 2 { // BlockMeta marshaled at least one field
		out = append(out, ',')
	}
	out = append(out, `"compactedTime":`...)
	out = append(out, ct...)
	out = append(out, '}')
	return out, nil
}
