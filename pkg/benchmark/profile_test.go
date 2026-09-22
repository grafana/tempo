package benchmark

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

func validProfile() *BlockProfile {
	meta := backend.NewBlockMeta("test-tenant", uuid.MustParse("00000000-0000-0000-0000-00000000beef"), "vParquet5")
	meta.StartTime = time.Unix(1000, 0).UTC()
	meta.EndTime = time.Unix(4600, 0).UTC()
	meta.TotalObjects = 10
	meta.TotalRecords = 3

	return &BlockProfile{
		SchemaVersion: ProfileSchemaVersion,
		GeneratedAt:   time.Unix(5000, 0).UTC(),
		GeneratedBy:   BuildInfo{TempoVersion: "0.0.0-test", GitSHA: "deadbeef"},
		Block:         meta,
		RowGroups:     3,
		TraceIDs: TraceIDProfile{
			Mode:    TraceIDModeSample,
			Present: []string{"0102030405060708090a0b0c0d0e0f10"},
			Absent:  []string{strings.Repeat("00", 16)},
		},
	}
}

func TestProfileRoundTrip(t *testing.T) {
	want := validProfile()

	var buf bytes.Buffer
	require.NoError(t, want.Write(&buf))

	got, err := LoadProfile(&buf)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestLoadProfileRejectsOtherSchemaVersion(t *testing.T) {
	p := validProfile()
	p.SchemaVersion = ProfileSchemaVersion + 1

	var buf bytes.Buffer
	require.NoError(t, p.Write(&buf))

	_, err := LoadProfile(&buf)
	require.ErrorContains(t, err, "schema version")
}

func TestProfileValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*BlockProfile)
		wantErr string
	}{
		{"no block", func(p *BlockProfile) { p.Block = nil }, "no block metadata"},
		{"no row groups", func(p *BlockProfile) { p.RowGroups = 0 }, "reports 0 row groups"},
		{"unknown id mode", func(p *BlockProfile) { p.TraceIDs.Mode = "some" }, "unknown trace ID mode"},
		{"short trace id", func(p *BlockProfile) { p.TraceIDs.Present = []string{"0102"} }, "is 4 characters"},
		{"empty trace id", func(p *BlockProfile) { p.TraceIDs.Absent = []string{""} }, "is 0 characters"},
		{
			name:    "non-hex trace id",
			mutate:  func(p *BlockProfile) { p.TraceIDs.Present = []string{strings.Repeat("z", 32)} },
			wantErr: "invalid trace ID",
		},
		{
			name:    "mode all with embedded ids",
			mutate:  func(p *BlockProfile) { p.TraceIDs.Mode = TraceIDModeAll },
			wantErr: "present IDs are embedded",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := validProfile()
			tc.mutate(p)
			require.ErrorContains(t, p.Validate(), tc.wantErr)
		})
	}

	require.NoError(t, validProfile().Validate())
}
