package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

// The BlocksKey value is base64url of a version byte followed by protobuf fields, repeated per block in order:
//
//	1: bytes  backend.BlockMeta without dedicated columns
//	2: bytes  dedicated columns JSON, written the first time a distinct set is used
//	3: uint64 1-based index of the block's dedicated columns in the order field 2 was written, 0 for none
//
// Dedicated columns repeat across a tenant's blocks, sending them once per job keeps encoding and decoding cheap.
const (
	blocksVersion = 1

	blocksMetaField       = 1
	blocksColumnsField    = 2
	blocksColumnsIdxField = 3
)

// EncodeTraceByIDBlocks encodes the blocks a trace by id job must search into the BlocksKey param value.
func EncodeTraceByIDBlocks(metas []*backend.BlockMeta) (string, error) {
	b := []byte{blocksVersion}
	columnsIdx := map[uint64]uint64{}

	for _, m := range metas {
		stripped := *m
		stripped.DedicatedColumns = nil
		mb, err := stripped.Marshal()
		if err != nil {
			return "", fmt.Errorf("error marshalling block %s: %w", m.BlockID, err)
		}
		b = protowire.AppendTag(b, blocksMetaField, protowire.BytesType)
		b = protowire.AppendBytes(b, mb)

		var idx uint64
		if len(m.DedicatedColumns) > 0 {
			hash := m.DedicatedColumns.Hash()
			var ok bool
			if idx, ok = columnsIdx[hash]; !ok {
				cb, err := m.DedicatedColumns.Marshal()
				if err != nil {
					return "", fmt.Errorf("error marshalling dedicated columns for block %s: %w", m.BlockID, err)
				}
				b = protowire.AppendTag(b, blocksColumnsField, protowire.BytesType)
				b = protowire.AppendBytes(b, cb)

				idx = uint64(len(columnsIdx)) + 1
				columnsIdx[hash] = idx
			}
		}
		b = protowire.AppendTag(b, blocksColumnsIdxField, protowire.VarintType)
		b = protowire.AppendVarint(b, idx)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ParseTraceByIDBlocks returns the blocks sent by the query-frontend, or nil if the request has no BlocksKey param.
// A present but empty list returns a non-nil empty slice.
func ParseTraceByIDBlocks(r *http.Request) ([]*backend.BlockMeta, error) {
	v, ok := extractQueryParam(r.URL.Query(), BlocksKey)
	if !ok {
		return nil, nil
	}

	metas, err := decodeTraceByIDBlocks(v)
	if err != nil {
		return nil, fmt.Errorf("invalid value for blocks: %w", err)
	}

	return metas, nil
}

func decodeTraceByIDBlocks(v string) ([]*backend.BlockMeta, error) {
	b, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 || b[0] != blocksVersion {
		return nil, errors.New("unsupported version")
	}
	b = b[1:]

	metas := []*backend.BlockMeta{}
	var columns []backend.DedicatedColumns
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		b = b[n:]

		switch {
		case num == blocksMetaField && typ == protowire.BytesType:
			mb, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]

			m := &backend.BlockMeta{}
			if err := m.Unmarshal(mb); err != nil {
				return nil, err
			}
			metas = append(metas, m)
		case num == blocksColumnsField && typ == protowire.BytesType:
			cb, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]

			var cols backend.DedicatedColumns
			if err := cols.Unmarshal(cb); err != nil {
				return nil, err
			}
			columns = append(columns, cols)
		case num == blocksColumnsIdxField && typ == protowire.VarintType:
			idx, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]

			if len(metas) == 0 || idx > uint64(len(columns)) {
				return nil, fmt.Errorf("invalid dedicated columns index %d", idx)
			}
			if idx > 0 {
				metas[len(metas)-1].DedicatedColumns = columns[idx-1]
			}
		default:
			return nil, fmt.Errorf("unexpected field %d", num)
		}
	}

	return metas, nil
}
