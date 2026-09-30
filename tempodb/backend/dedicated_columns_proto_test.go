package backend

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDedicatedColumnsProtoUnmarshal(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want DedicatedColumns
	}{
		{"empty", "", nil},
		{"null", "null", nil},
		{"array", "[]", DedicatedColumns{}},
		{"defaults", `[{"n":"http.method"}]`, DedicatedColumns{{Scope: "span", Type: "string", Name: "http.method"}}},
		{"whitespace", " \n [ {\"n\": \"http.method\"} ] \t", DedicatedColumns{{Scope: "span", Type: "string", Name: "http.method"}}},
		{"options", `[{"s":"resource","n":"tags","t":"string","o":["array"]}]`, DedicatedColumns{{Scope: "resource", Type: "string", Name: "tags", Options: DedicatedColumnOptions{"array"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dedicatedColumnsCache.InvalidateAll()
			for range 2 {
				var got DedicatedColumns
				require.NoError(t, got.Unmarshal([]byte(tc.data)))
				require.Equal(t, tc.want, got)
			}
		})
	}
}

func TestDedicatedColumnsProtoUnmarshalRejectsInvalid(t *testing.T) {
	valid := `[{"n":"http.method"}]`
	for _, data := range []string{valid + "x", valid + "[]", valid[:len(valid)-1], `{}`, `[{"n":42}]`, " "} {
		t.Run(data, func(t *testing.T) {
			dedicatedColumnsCache.InvalidateAll()
			var warm DedicatedColumns
			require.NoError(t, warm.Unmarshal([]byte(valid)))
			for range 2 {
				var got DedicatedColumns
				require.Error(t, got.Unmarshal([]byte(data)))
			}
		})
	}
}

func TestDedicatedColumnsProtoUnmarshalReusedReceiver(t *testing.T) {
	for _, data := range []string{"", "null", "[]", `[{"n":"http.method"}]`} {
		t.Run(data, func(t *testing.T) {
			dedicatedColumnsCache.InvalidateAll()
			for range 2 {
				got := DedicatedColumns{{Scope: "resource", Type: "int", Name: "old"}}
				require.NoError(t, got.Unmarshal([]byte(data)))
				switch data {
				case "", "null":
					require.Equal(t, DedicatedColumns{{Scope: "resource", Type: "int", Name: "old"}}, got)
				case "[]":
					require.Equal(t, DedicatedColumns{}, got)
				default:
					require.Equal(t, DedicatedColumns{{Scope: "resource", Type: "int", Name: "http.method"}}, got)
				}
			}
		})
	}
}

func TestDedicatedColumnsProtoUnmarshalOwnsCacheKey(t *testing.T) {
	dedicatedColumnsCache.InvalidateAll()
	data := []byte(`[{"n":"http.method"}]`)
	var got DedicatedColumns
	require.NoError(t, got.Unmarshal(data))
	clear(data)
	cached, ok := getDedicatedColumnsFromCache([]byte(`[{"n":"http.method"}]`))
	require.True(t, ok)
	require.Equal(t, got, cached)
	var again DedicatedColumns
	require.NoError(t, again.Unmarshal([]byte(`[{"n":"http.method"}]`)))
	require.Equal(t, DedicatedColumns{{Scope: "span", Type: "string", Name: "http.method"}}, again)
}

func TestDedicatedColumnsProtoUnmarshalConcurrent(t *testing.T) {
	dedicatedColumnsCache.InvalidateAll()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				var got DedicatedColumns
				if err := got.Unmarshal([]byte(`[{"n":"http.method"}]`)); err != nil {
					t.Error(err)
					return
				}
				if len(got) != 1 || got[0].Name != "http.method" || got[0].Scope != "span" || got[0].Type != "string" {
					t.Errorf("unexpected columns: %v", got)
					return
				}
			}
		})
	}
	wg.Wait()
}
