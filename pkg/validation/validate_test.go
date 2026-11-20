package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidTraceID(t *testing.T) {
	tests := []struct {
		name string
		id   []byte
		want bool
	}{
		{"nil", nil, false},
		{"empty", []byte{}, false},
		{"one byte short", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, false},
		{"too long", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}, false},
		{"all zero", make([]byte, 16), false},
		{"valid", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ValidTraceID(tt.id))
		})
	}
}

func TestValidSpanID(t *testing.T) {
	tests := []struct {
		name string
		id   []byte
		want bool
	}{
		{"nil", nil, false},
		{"empty", []byte{}, false},
		{"one byte short", []byte{1, 2, 3, 4, 5, 6, 7}, false},
		{"too long", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, false},
		{"trace id length", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, false},
		{"all zero", make([]byte, 8), false},
		{"valid", []byte{1, 2, 3, 4, 5, 6, 7, 8}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ValidSpanID(tt.id))
		})
	}
}
