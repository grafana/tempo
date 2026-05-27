package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("FOO", "bar")
	t.Setenv("EMPTY", "")

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "no expansion", in: "hello world", want: "hello world"},
		{name: "simple $VAR", in: "x=$FOO", want: "x=bar"},
		{name: "braced ${VAR}", in: "x=${FOO}", want: "x=bar"},
		{name: "host:port URL", in: "${FOO}:9095", want: "bar:9095"},
		{name: "unknown var expands empty", in: "${UNSET}", want: ""},
		{name: "empty var expands empty", in: "${EMPTY}", want: ""},

		{name: "default syntax rejected", in: "${MISSING:-x}", wantErr: true},
		{name: "error syntax rejected", in: "${MISSING:?msg}", wantErr: true},
		{name: "alternative syntax rejected", in: "${FOO:+x}", wantErr: true},
		{name: "uppercase syntax rejected", in: "${FOO^^}", wantErr: true},
		{name: "length syntax rejected", in: "${#FOO}", wantErr: true},
		{name: "replace syntax rejected", in: "${FOO/x/y}", wantErr: true},
		{name: "substring syntax rejected", in: "${FOO:1:2}", wantErr: true},
		{name: "unclosed brace rejected", in: "${FOO", wantErr: true},
		{name: "$$ escape rejected", in: "literal$$dollar", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandEnv(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
