package backend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDedicatedColumnsEncodedSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		cols DedicatedColumns
	}{
		{"nil", nil},
		{"empty", DedicatedColumns{}},
		{"zero value", DedicatedColumns{{}}},
		{"defaults", DefaultDedicatedColumns()},
		{"options", DedicatedColumns{{Name: "attribute", Scope: DedicatedColumnScopeEvent, Type: DedicatedColumnTypeInt, Options: DedicatedColumnOptions{DedicatedColumnOptionArray, DedicatedColumnOptionBlob}}}},
		{"escaping", DedicatedColumns{{Name: "<>&\"\\\n☃"}}},
		{"32-byte name", DedicatedColumns{{Name: strings.Repeat("x", 32)}}},
		{"33-byte name", DedicatedColumns{{Name: strings.Repeat("x", 33)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.cols.Marshal()
			require.NoError(t, err)
			require.Equal(t, len(data), tc.cols.Size())
		})
	}
}

func FuzzDedicatedColumnsEncodedSize(f *testing.F) {
	f.Add("http.method", "span", "string", "", false)
	f.Add("☃\"\\\n", "resource", "int", "blob", true)
	f.Add(string([]byte{0xff}), "event", "", "array", true)
	f.Fuzz(func(t *testing.T, name, scope, kind, option string, options bool) {
		cols := DedicatedColumns{{Name: name, Scope: DedicatedColumnScope(scope), Type: DedicatedColumnType(kind)}}
		if options {
			cols[0].Options = DedicatedColumnOptions{DedicatedColumnOption(option), ""}
		}
		cols = append(cols, DefaultDedicatedColumns()...)
		data, err := cols.Marshal()
		require.NoError(t, err)
		require.Equal(t, len(data), cols.Size())
	})
}
