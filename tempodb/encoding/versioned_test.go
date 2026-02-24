package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromVersionErrors(t *testing.T) {
	encoding, err := FromVersion("definitely-not-a-real-version")
	assert.Error(t, err)
	assert.Nil(t, encoding)
}

func TestCoalesceVersion(t *testing.T) {
	defaultVer := DefaultEncoding().Version()

	tests := []struct {
		name        string
		versions    []string
		expectedVer string
		expectErr   bool
	}{
		{
			name:        "no versions returns default",
			versions:    nil,
			expectedVer: defaultVer,
		},
		{
			name:        "empty strings return default",
			versions:    []string{"", ""},
			expectedVer: defaultVer,
		},
		{
			name:        "single version",
			versions:    []string{LatestEncoding().Version()},
			expectedVer: LatestEncoding().Version(),
		},
		{
			name:        "later non-empty wins",
			versions:    []string{"vParquet4", "vParquet5"},
			expectedVer: "vParquet5",
		},
		{
			name:        "empty string does not override",
			versions:    []string{"vParquet5", ""},
			expectedVer: "vParquet5",
		},
		{
			name:      "invalid version returns error",
			versions:  []string{"invalid"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := CoalesceVersion(tt.versions...)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedVer, enc.Version())
		})
	}
}

func TestAllVersions(t *testing.T) {
	for _, v := range AllEncodings() {
		encoding, err := FromVersion(v.Version())

		require.Equal(t, v.Version(), encoding.Version())
		require.NoError(t, err)
	}
}
