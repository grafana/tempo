package wal

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

func TestFullFilename(t *testing.T) {
	tests := []struct {
		name     string
		b        *block
		expected string
	}{
		{
			name: "ez-mode",
			b: &block{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone),
				filepath: "/blerg",
			},
			expected: "/blerg/123e4567-e89b-12d3-a456-426614174000:foo",
		},
		{
			name: "nopath",
			b: &block{
				meta:     backend.NewBlockMeta("foo", uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), "v1", backend.EncNone),
				filepath: "",
			},
			expected: "123e4567-e89b-12d3-a456-426614174000:foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.b.fullFilename())
		})
	}
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		expectUUID   uuid.UUID
		expectTenant string
		expectError  bool
	}{
		{
			name:         "ez-mode",
			filename:     "123e4567-e89b-12d3-a456-426614174000:foo",
			expectUUID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectTenant: "foo",
		},
		{
			name:        "path fails",
			filename:    "/blerg/123e4567-e89b-12d3-a456-426614174000:foo",
			expectError: true,
		},
		{
			name:        "no :",
			filename:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: true,
		},
		{
			name:        "empty string",
			filename:    "",
			expectError: true,
		},
		{
			name:        "bad uuid",
			filename:    "123e4:foo",
			expectError: true,
		},
		{
			name:        "no tenant",
			filename:    "123e4567-e89b-12d3-a456-426614174000:",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualUUID, actualTenant, err := parseFilename(tc.filename)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectUUID, actualUUID)
			assert.Equal(t, tc.expectTenant, actualTenant)
		})
	}
}
