package backend

import (
	"bytes"
	"encoding/json"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDedicatedColumnLayoutEncoding(t *testing.T) {
	for _, cols := range []DedicatedColumns{
		nil,
		{},
		{{}},
		DefaultDedicatedColumns(),
		{{Name: "<>&\"\\\n☃"}},
		{{Name: string([]byte{255, 0, 254})}},
		{{Name: "options", Scope: "event", Type: "int", Options: DedicatedColumnOptions{"array", "blob"}}},
	} {
		t.Run(stringMustJSON(t, cols), func(t *testing.T) {
			layout := NewDedicatedColumnLayout(cols)
			want, err := cols.Marshal()
			require.NoError(t, err)
			require.Equal(t, len(want), layout.Size())
			got := make([]byte, layout.Size())
			n, err := layout.MarshalTo(got)
			require.NoError(t, err)
			require.Equal(t, len(want), n)
			require.True(t, bytes.Equal(want, got))
			var decoded DedicatedColumnLayout
			require.NoError(t, decoded.Unmarshal(got))
			roundtrip := make([]byte, decoded.Size())
			_, err = decoded.MarshalTo(roundtrip)
			require.NoError(t, err)
			require.Equal(t, got, roundtrip)
		})
	}
}

func stringMustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func TestDedicatedColumnLayoutOwnsColumns(t *testing.T) {
	cols := DedicatedColumns{{Name: "original", Options: DedicatedColumnOptions{"array"}}}
	layout := NewDedicatedColumnLayout(cols)
	expected, err := layout.MarshalJSON()
	require.NoError(t, err)
	cols[0].Name = "changed"
	cols[0].Options[0] = "blob"
	exposed := layout.Columns()
	exposed[0].Name = "changed"
	exposed[0].Options[0] = "blob"
	for i := range layout.Len() {
		col := layout.At(i)
		col.Column().Options[0] = "blob"
	}
	data, err := layout.MarshalJSON()
	require.NoError(t, err)
	clear(data)
	got, err := layout.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, expected, got)
	copied := layout
	require.NoError(t, copied.UnmarshalJSON([]byte(`[{"n":"replacement"}]`)))
	got, err = layout.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, expected, got)
	require.NotEqual(t, layout.Columns(), copied.Columns())
	require.True(t, layout.Equal(NewDedicatedColumnLayout(layout.Columns())))
}

func TestDedicatedColumnLayoutDecode(t *testing.T) {
	for _, input := range []string{"", "null", "[]", `[{"n":"new"}]`, `[{"n":"new","o":["blob"]}]`} {
		t.Run(input, func(t *testing.T) {
			for range 2 {
				original := DedicatedColumns{{Scope: "resource", Type: "int", Name: "old", Options: DedicatedColumnOptions{"array"}}}
				got := NewDedicatedColumnLayout(original)
				_ = got.Size()
				type plain DedicatedColumns
				expected := cloneColumns(original)
				if input != "" && input != "null" {
					require.NoError(t, json.Unmarshal([]byte(input), (*plain)(&expected)))
				}
				require.NoError(t, got.Unmarshal([]byte(input)))
				require.True(t, slices.EqualFunc(expected, got.Columns(), func(a, b DedicatedColumn) bool {
					return a.Scope == b.Scope && a.Type == b.Type && a.Name == b.Name && slices.Equal(a.Options, b.Options)
				}))
			}
		})
	}
	for _, input := range []string{`[{"n":42}]`, `[{"n":"x"}]x`, `{}`, `[`} {
		var layout DedicatedColumnLayout
		require.Error(t, layout.Unmarshal([]byte(input)))
	}
}

func TestDedicatedColumnLayoutConcurrent(t *testing.T) {
	layout := NewDedicatedColumnLayout(DefaultDedicatedColumns())
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				data := make([]byte, layout.Size())
				n, err := layout.MarshalTo(data)
				if err != nil || n != len(data) {
					t.Errorf("marshal: %v, bytes=%d", err, n)
				}
				var decoded DedicatedColumnLayout
				if err := decoded.Unmarshal(data); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
}
